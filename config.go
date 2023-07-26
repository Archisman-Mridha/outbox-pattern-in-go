package main

type (
	Config struct {
		Sources *Sources `yaml:"sources"`
		Sink *Sink `yaml:"sink"` // Currently only RabbitMQ is supported.
	}

	Sources struct {
		Postgres *Postgres `yaml:"postgres"`
		Redis *Redis `yaml:"redis"`
	}

	Postgres struct {
		Uri string `yaml:"uri"`
		BatchSize int `yaml:"batch_size"`
	}

	Redis struct {
		Uri string `yaml:"uri"`
		Password string `yaml:"password"`
		Db int `yaml:"db"`

		BatchSize int `yaml:"batch_size"`
	}

	Sink struct {
		Uri string `yaml:"uri"`
		Queue string `yaml:"queue"`
	}
)