package particlerepository

// GetPublishedParticles — все опубликованные частицы
func GetPublishedParticles() []Particle {
	var particles []Particle
	DB.Where("status = ?", StatusPublished).Order("mass ASC").Find(&particles)
	return particles
}

// GetPublishedParticleByID — опубликованная частица по ID
func GetPublishedParticleByID(id int) *Particle {
	var particle Particle
	result := DB.Where("id = ? AND status = ?", id, StatusPublished).First(&particle)
	if result.Error != nil {
		return nil
	}
	return &particle
}

// GetParticleByID — любая частица по ID
func GetParticleByID(id int) *Particle {
	var particle Particle
	result := DB.First(&particle, id)
	if result.Error != nil {
		return nil
	}
	return &particle
}

// GetDraftParticle — черновик
func GetDraftParticle() *Particle {
	var particle Particle
	result := DB.Where("status = ?", StatusDraft).First(&particle)
	if result.Error != nil {
		return nil
	}
	return &particle
}

// FilterParticlesByMass — фильтрация по массе
func FilterParticlesByMass(minMass float64) []Particle {
	var particles []Particle
	DB.Where("status = ? AND mass >= ?", StatusPublished, minMass).
		Order("mass ASC").
		Find(&particles)
	return particles
}

// GetParticleLikesCount — количество лайков частицы
func GetParticleLikesCount(particleID uint) int {
	var count int64
	DB.Model(&Like{}).Where("particle_id = ?", particleID).Count(&count)
	return int(count)
}

// CreateParticle — создать новую частицу
func CreateParticle(particle *Particle) error {
	return DB.Create(particle).Error
}

// UpdateParticle — обновить частицу (для публикации)
func UpdateParticle(id uint, updates map[string]interface{}) error {
	return DB.Model(&Particle{}).Where("id = ?", id).Updates(updates).Error
}

// PublishParticle — опубликовать частицу (сменить статус)
func PublishParticle(id uint) error {
	return DB.Model(&Particle{}).Where("id = ?", id).Update("status", StatusPublished).Error
}

// DeleteParticleSQL — логическое удаление через SQL UPDATE (без ORM)
func DeleteParticleSQL(id uint) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec("UPDATE particles SET status = 'deleted' WHERE id = $1", id)
	return err
}
