package particlerepository

import "sync"

const (
	StatusPublished = "published"
	StatusDraft     = "draft"
	StatusDeleted   = "deleted"
)

type Particle struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Mass            float64 `json:"mass"`
	Spin            string  `json:"spin"`
	Wavelength      float64 `json:"wavelength"`
	Status          string  `json:"status"`
	ImageKey        string  `json:"image_key"`
	VideoKey        string  `json:"video_key"`
	Likes           []int   `json:"likes"`
	Description     string  `json:"description"`
	FullDescription string  `json:"full_description"`
}

var particlesMu sync.RWMutex

var Particles = []Particle{
	{
		ID:              1,
		Name:            "Электрон",
		Mass:            9.11e-31,
		Spin:            "1/2",
		Wavelength:      3.32e-10,
		Status:          StatusPublished,
		ImageKey:        "electron.jpg",
		VideoKey:        "electron.mp4",
		Likes:           []int{1, 2, 3, 5, 8, 13, 21},
		Description:     "Электрон — элементарная частица с массой ≈ 9.11 × 10⁻³¹ кг.",
		FullDescription: "Электрон — стабильная элементарная частица из семейства лептонов. Он обладает отрицательным электрическим зарядом и участвует в электромагнитном и слабом взаимодействиях. Электроны образуют оболочки атомов, а их волновые свойства становятся особенно заметны при движении с большим импульсом.",
	},
	{
		ID:              2,
		Name:            "Протон",
		Mass:            1.67e-27,
		Spin:            "1/2",
		Wavelength:      3.96e-13,
		Status:          StatusPublished,
		ImageKey:        "proton.jpg",
		VideoKey:        "proton.mp4",
		Likes:           []int{4, 7, 10},
		Description:     "Протон — стабильная частица с массой ≈ 1.67 × 10⁻²⁷ кг.",
		FullDescription: "Протон — положительно заряженная частица, входящая в состав атомных ядер. Он относится к адронам и состоит из двух u-кварков и одного d-кварка, связанных сильным взаимодействием.",
	},
	{
		ID:              3,
		Name:            "Нейтрино",
		Mass:            0.1e-31,
		Spin:            "1/2",
		Wavelength:      6.626e-34,
		Status:          StatusDraft,
		ImageKey:        "neutrino.jpg",
		VideoKey:        "neutrino.mp4",
		Likes:           []int{},
		Description:     "Нейтрино — легчайшая частица с массой почти 0.",
		FullDescription: "Нейтрино — электрически нейтральная элементарная частица с очень малой массой. Крайне слабо взаимодействует с веществом.",
	},
	{
		ID:              4,
		Name:            "Мюон",
		Mass:            1.88e-28,
		Spin:            "1/2",
		Wavelength:      3.52e-12,
		Status:          StatusDeleted,
		ImageKey:        "muon.jpg",
		VideoKey:        "muon.mp4",
		Likes:           []int{6, 9},
		Description:     "Мюон — нестабильная частица, тяжелее электрона.",
		FullDescription: "Мюон — элементарная частица из семейства лептонов. Возникает при взаимодействии космических лучей с атмосферой Земли.",
	},
	{
		ID:              5,
		Name:            "Нейтрон",
		Mass:            1.675e-27,
		Spin:            "1/2",
		Wavelength:      3.95e-13,
		Status:          StatusPublished,
		ImageKey:        "neutron.jpg",
		VideoKey:        "neutron.mp4",
		Likes:           []int{3, 7},
		Description:     "Нейтрон — нейтральная частица, входящая в состав ядер атомов.",
		FullDescription: "Нейтрон — электрически нейтральная частица, которая вместе с протонами образует атомные ядра.",
	},
	{
		ID:              7,
		Name:            "Позитрон",
		Mass:            9.11e-31,
		Spin:            "1/2",
		Wavelength:      2.909e-10,
		Status:          StatusPublished,
		ImageKey:        "positron.jpg",
		VideoKey:        "positron.mp4",
		Likes:           []int{1, 8},
		Description:     "Позитрон — античастица электрона.",
		FullDescription: "Позитрон — античастица электрона с такой же массой и спином, но с положительным электрическим зарядом.",
	},
}

func GetPublishedParticles() []Particle {
	particlesMu.RLock()
	defer particlesMu.RUnlock()

	var result []Particle
	for _, p := range Particles {
		if p.Status == StatusPublished {
			result = append(result, p)
		}
	}
	return result
}

func GetPublishedParticleByID(id int) *Particle {
	particlesMu.RLock()
	defer particlesMu.RUnlock()

	for i := range Particles {
		if Particles[i].ID == id && Particles[i].Status == StatusPublished {
			p := Particles[i]
			return &p
		}
	}
	return nil
}

func GetParticleByID(id int) *Particle {
	particlesMu.RLock()
	defer particlesMu.RUnlock()

	for i := range Particles {
		if Particles[i].ID == id {
			p := Particles[i]
			return &p
		}
	}
	return nil
}

func GetDraftParticle() *Particle {
	particlesMu.RLock()
	defer particlesMu.RUnlock()

	for i := range Particles {
		if Particles[i].Status == StatusDraft {
			p := Particles[i]
			return &p
		}
	}
	return nil
}

func FilterParticlesByMass(minMass float64) []Particle {
	particlesMu.RLock()
	defer particlesMu.RUnlock()

	var result []Particle
	for _, p := range Particles {
		if p.Status == StatusPublished && p.Mass >= minMass {
			result = append(result, p)
		}
	}
	return result
}

func GetParticleLikesCount(likes []int) int {
	return len(likes)
}
