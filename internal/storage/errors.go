package storage

import "errors"

// ErrNotFound — хранилище не нашло запись. Домен не зависит от ORM,
// поэтому «не найдено» передаётся через эту ошибку, а не gorm.ErrRecordNotFound.
var ErrNotFound = errors.New("запись не найдена")

// ErrDuplicatedKey — нарушено уникальное ограничение (например, email уже занят).
var ErrDuplicatedKey = errors.New("нарушено уникальное ограничение")
