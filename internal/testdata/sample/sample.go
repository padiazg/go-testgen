package sample

type Config struct {
	Url string
}

type Engine struct {
	config *Config
}

func New(cfg *Config) *Engine {
	return &Engine{config: cfg}
}

func (e *Engine) Start() error {
	return nil
}

func (e *Engine) Notify(msg string) {
}
