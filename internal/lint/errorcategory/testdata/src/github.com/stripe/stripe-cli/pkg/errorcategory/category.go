package errorcategory

type Category string

const (
	UserInput Category = "user_input"
	Auth      Category = "auth"
	Network   Category = "network"
)

func New(Category, string) error { return nil }

func Errorf(Category, string, ...any) error { return nil }

func With(error, Category) error { return nil }
