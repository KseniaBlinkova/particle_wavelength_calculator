package particle

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// translitTable — транслитерация кириллицы в латиницу для имён файлов.
var translitTable = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// extensionByContentType — расширение файла определяется по его содержимому,
// чтобы имя в баке и в поле модели совпадало с типом файла.
var extensionByContentType = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"video/mp4":  ".mp4",
	"video/webm": ".webm",
}

// newFileKey — генерирует имя файла на латинице: <kind>_<латиница названия>_<дата>_<случайный суффикс>.
// Ключ сохраняется в поле модели (image_key / video_key), а файл кладётся в MinIO.
func newFileKey(kind, contentType, originalName string, at time.Time) string {
	extension, ok := extensionByContentType[contentType]
	if !ok {
		extension = strings.ToLower(filepath.Ext(originalName))
	}

	base := latinize(strings.TrimSuffix(filepath.Base(originalName), filepath.Ext(originalName)))
	if len(base) > 32 {
		base = base[:32]
	}
	return fmt.Sprintf("%s_%s_%s_%s%s", kind, base, at.Format("20060102_150405"), randomSuffix(), extension)
}

// latinize — переводит имя файла в нижний регистр латиницы: кириллица
// транслитерируется, пробелы и знаки заменяются на дефис, остальное отбрасывается.
func latinize(value string) string {
	var builder strings.Builder
	for _, symbol := range value {
		if latin, ok := translitTable[unicode.ToLower(symbol)]; ok {
			builder.WriteString(latin)
			continue
		}
		if isLatinAllowed(symbol) {
			builder.WriteRune(unicode.ToLower(symbol))
			continue
		}
		if unicode.IsSpace(symbol) || symbol == '_' || symbol == '.' || symbol == '(' || symbol == ')' {
			builder.WriteRune('-')
		}
	}

	cleaned := strings.Join(strings.FieldsFunc(builder.String(), func(r rune) bool { return r == '-' }), "-")
	return strings.ToLower(cleaned)
}

func isLatinAllowed(symbol rune) bool {
	return symbol >= 'a' && symbol <= 'z' || symbol >= '0' && symbol <= '9'
}

func randomSuffix() string {
	buffer := make([]byte, 3)
	if _, err := rand.Read(buffer); err != nil {
		return "000000"
	}
	return hex.EncodeToString(buffer)
}
