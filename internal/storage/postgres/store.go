package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"particleapp/internal/auth"
	"particleapp/internal/models"
)

type Store struct {
	db *gorm.DB
}

func New(dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	// В ранних версиях лабораторной работы ограничения могли
	// отсутствовать, поэтому в старой базе могут быть дубли лайков
	// или несколько черновиков. Исправляем их до того, как AutoMigrate
	// начнёт создавать уникальные индексы.
	if err := normalizeLegacyData(ctx, db); err != nil {
		return nil, fmt.Errorf("подготовка старой схемы: %w", err)
	}

	if err := db.WithContext(ctx).AutoMigrate(&models.User{}, &models.Particle{}, &models.Like{}); err != nil {
		return nil, fmt.Errorf("миграция схемы: %w", err)
	}
	if err := ensureCurrentUser(ctx, db); err != nil {
		return nil, fmt.Errorf("создание пользователя лабораторной работы: %w", err)
	}

	return &Store{db: db}, nil
}

// normalizeLegacyData делает повторный запуск миграций безопасным
// для базы, созданной предыдущей версией проекта.
func normalizeLegacyData(ctx context.Context, db *gorm.DB) error {
	if db.Migrator().HasTable(&models.Like{}) {
		// Оставляем самый ранний лайк для каждой пары user_id/particle_id.
		if err := db.WithContext(ctx).Exec(`
			DELETE FROM likes AS duplicate
			USING likes AS original
			WHERE duplicate.user_id = original.user_id
			  AND duplicate.particle_id = original.particle_id
			  AND duplicate.id > original.id
		`).Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasTable(&models.Particle{}) {
		// Оставляем последний черновик, а более старые мягко удаляем.
		if err := db.WithContext(ctx).Exec(`
			UPDATE particles AS older
			SET status = 'deleted',
			    completed_at = COALESCE(older.completed_at, CURRENT_TIMESTAMP)
			WHERE older.status = 'draft'
			  AND EXISTS (
				SELECT 1
				FROM particles AS newer
				WHERE newer.creator_id = older.creator_id
				  AND newer.status = 'draft'
				  AND newer.id > older.id
			  )
		`).Error; err != nil {
			return err
		}
	}

	return nil
}

// ensureCurrentUser гарантирует, что singleton-пользователь существует даже
// если база была создана только через GORM, без запуска SQL-миграции.
func ensureCurrentUser(ctx context.Context, db *gorm.DB) error {
	var current models.User
	err := db.WithContext(ctx).First(&current, auth.CurrentUserID).Error
	if err == nil {
		return syncUserSequence(ctx, db)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	current = models.User{
		ID:       auth.CurrentUserID,
		Username: "ksenia",
		Email:    "ksenia@mail.ru",
	}
	if err := db.WithContext(ctx).Create(&current).Error; err != nil {
		return err
	}

	// PostgreSQL не изменяет sequence после INSERT с явно заданным id.
	return syncUserSequence(ctx, db)
}

func syncUserSequence(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Exec(`
		SELECT setval(
			pg_get_serial_sequence('users', 'id'),
			(SELECT MAX(id) FROM users),
			true
		)
	`).Error
}

func (s *Store) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
