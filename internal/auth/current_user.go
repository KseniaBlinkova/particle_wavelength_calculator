package auth

const CurrentUserID uint = 1

type singleton struct {
	ID uint
}

var instance = &singleton{ID: CurrentUserID}

// Get — функция-singleton: возвращает единственный экземпляр текущего
// пользователя, заданного константой CurrentUserID.
func Get() *singleton {
	return instance
}
