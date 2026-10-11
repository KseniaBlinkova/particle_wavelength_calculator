package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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

	// Услуги, опубликованные до появления formed_at, получают дату
	// формирования, равную дате создания: другой информации о них нет.
	if err := db.WithContext(ctx).Model(&models.Particle{}).
		Where("status = ? AND formed_at IS NULL", models.StatusPublished).
		Update("formed_at", gorm.Expr("created_at")).Error; err != nil {
		return nil, fmt.Errorf("заполнение formed_at: %w", err)
	}
	return &Store{db: db}, nil
}

// normalizeLegacyData делает повторный запуск миграций безопасным
// для базы, созданной предыдущей версией проекта.
func normalizeLegacyData(ctx context.Context, db *gorm.DB) error {
	// Пользователь теперь состоит только из логина и пароля: прежний
	// username становится логином, email и password_hash удаляются.
	if db.Migrator().HasTable(&models.User{}) {
		migrator := db.WithContext(ctx).Migrator()
		if migrator.HasColumn(&models.User{}, "username") && !migrator.HasColumn(&models.User{}, "login") {
			if err := migrator.RenameColumn(&models.User{}, "username", "login"); err != nil {
				return err
			}
		}
		for _, column := range []string{"email", "password_hash"} {
			if migrator.HasColumn(&models.User{}, column) {
				if err := migrator.DropColumn(&models.User{}, column); err != nil {
					return err
				}
			}
		}
	}

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

func (s *Store) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
