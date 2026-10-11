package particle

import (
	"bytes"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"particleapp/internal/apperror"
	"particleapp/internal/web"
)

const maxRequestBytes = 64 << 20

// Register — маршруты домена «услуга». Все адреса начинаются с /api.
func Register(mux *http.ServeMux, domain Domain) {
	mux.HandleFunc("GET /api/particles", func(w http.ResponseWriter, r *http.Request) {
		list(domain)(w, r)
	})
	mux.HandleFunc("GET /api/particles/feed", func(w http.ResponseWriter, r *http.Request) {
		feed(domain)(w, r)
	})
	mux.HandleFunc("GET /api/particles/draft", func(w http.ResponseWriter, r *http.Request) {
		draft(domain)(w, r)
	})
	mux.HandleFunc("POST /api/particles", func(w http.ResponseWriter, r *http.Request) {
		create(domain)(w, r)
	})
	mux.HandleFunc("PUT /api/particles/{id}/publish", func(w http.ResponseWriter, r *http.Request) {
		publish(domain)(w, r)
	})
	mux.HandleFunc("POST /api/particles/{id}/like", func(w http.ResponseWriter, r *http.Request) {
		like(domain)(w, r)
	})
	mux.HandleFunc("DELETE /api/particles/{id}", func(w http.ResponseWriter, r *http.Request) {
		remove(domain)(w, r)
	})
}

// list — GET /api/particles?min_mass=
func list(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		var filter Filter
		if raw := query.Get("min_mass"); raw != "" {
			minMass, err := strconv.ParseFloat(raw, 64)
			if err != nil || minMass < 0 || math.IsNaN(minMass) || math.IsInf(minMass, 0) {
				web.WriteError(w, apperror.Validation("Параметр min_mass должен быть неотрицательным числом"))
				return
			}
			filter.MinMass = &minMass
		}

		response, err := domain.List(r.Context(), filter)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, response)
	}
}

// feed — GET /api/particles/feed?id=&next=true
func feed(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		var id *uint
		if raw := query.Get("id"); raw != "" {
			parsed, err := strconv.ParseUint(raw, 10, 64)
			if err != nil || parsed == 0 {
				web.WriteError(w, apperror.Validation("Параметр id должен быть положительным числом"))
				return
			}
			value := uint(parsed)
			id = &value
		}

		if raw := query.Get("next"); raw != "" && raw != "true" && raw != "false" {
			web.WriteError(w, apperror.Validation("Параметр next принимает только true или false"))
			return
		}
		next := query.Get("next") == "true" && id != nil

		response, err := domain.Feed(r.Context(), id, next)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, response)
	}
}

// draft — GET /api/particles/draft
func draft(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, err := domain.Draft(r.Context())
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, response)
	}
}

// create — POST /api/particles, тело multipart/form-data с файлами image и video.
func create(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			web.WriteError(w, apperror.UnsupportedMediaType("Ожидается multipart/form-data с файлами image и video"))
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			web.WriteError(w, multipartError(err))
			return
		}
		defer r.MultipartForm.RemoveAll()

		request := CreateRequest{
			Name:            r.FormValue("name"),
			Spin:            r.FormValue("spin"),
			FullDescription: r.FormValue("full_description"),
		}
		if raw := r.FormValue("mass"); raw != "" {
			mass, err := strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(mass) || math.IsInf(mass, 0) {
				web.WriteError(w, apperror.Validation("Поле mass должно быть числом"))
				return
			}
			request.Mass = &mass
		}

		image, err := uploadedFile(r, "image")
		if err != nil {
			web.WriteError(w, err)
			return
		}
		video, err := uploadedFile(r, "video")
		if err != nil {
			closeUpload(image)
			web.WriteError(w, err)
			return
		}
		defer closeUpload(image)
		defer closeUpload(video)
		if image == nil || video == nil {
			web.WriteError(w, apperror.Validation("Поля image и video должны содержать файлы"))
			return
		}
		request.Image, request.Video = image, video

		response, err := domain.Create(r.Context(), request)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusCreated, response)
	}
}

// publish — PUT /api/particles/{id}/publish, смена статуса на published.
func publish(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := particleID(r)
		if err != nil {
			web.WriteError(w, err)
			return
		}

		var request PublishRequest
		if r.ContentLength > 0 {
			if err := web.DecodeJSON(r, &request); err != nil {
				web.WriteError(w, err)
				return
			}
		}

		response, err := domain.Publish(r.Context(), id, request)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, response)
	}
}

// like — POST /api/particles/{id}/like, тело {"liked": 0|1}
func like(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := particleID(r)
		if err != nil {
			web.WriteError(w, err)
			return
		}

		var request LikeRequest
		if err := web.DecodeJSON(r, &request); err != nil {
			web.WriteError(w, err)
			return
		}

		response, err := domain.Like(r.Context(), id, request.Liked)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, response)
	}
}

// remove — DELETE /api/particles/{id}, только soft delete.
func remove(domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := particleID(r)
		if err != nil {
			web.WriteError(w, err)
			return
		}

		if err := domain.Delete(r.Context(), id); err != nil {
			web.WriteError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func particleID(r *http.Request) (uint, error) {
	parsed, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || parsed == 0 {
		return 0, apperror.Validation("Некорректный идентификатор услуги")
	}
	return uint(parsed), nil
}

// uploadedFile — файл из формы. Тип определяется по содержимому, а не по
// расширению, которое прислал клиент.
func uploadedFile(r *http.Request, field string) (*UploadedFile, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, multipartError(err)
	}

	head := make([]byte, 512)
	read, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		file.Close()
		return nil, multipartError(err)
	}

	return &UploadedFile{
		Reader:       io.MultiReader(bytes.NewReader(head[:read]), file),
		OriginalName: header.Filename,
		Size:         header.Size,
		ContentType:  http.DetectContentType(head[:read]),
		closer:       file,
	}, nil
}

func closeUpload(file *UploadedFile) {
	if file != nil {
		_ = file.Close()
	}
}

func multipartError(err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return apperror.PayloadTooLarge("file_too_large", "Размер запроса больше допустимых 64 МБ")
	}
	if errors.Is(err, http.ErrMissingFile) {
		return apperror.Validation("Не найден файл в поле формы")
	}
	return apperror.Validation("Некорректное тело запроса: " + err.Error())
}
