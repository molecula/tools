package loader

import (
	"io"
	"time"

	"github.com/BurntSushi/toml"
)

// the toml package does not handle durations properly, so we have to create special handling for them.

type duration time.Duration

func (d *duration) UnmarshalText(text []byte) error {
	dr, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = duration(dr)
	return nil
}

func (d duration) MarshalText() (text []byte, err error) {
	return []byte(time.Duration(d).String()), nil
}

func DemoConfig() *Config {
	return &Config{
		Tasks: Tasks{
			{
				Connections: 10,
				Delay:       duration(time.Millisecond * 10),
				URL:         "http://localhost:10101/index/equipment/query",
				Query:       "TopN(model, n=5)",
			},
			{
				Connections: 1,
				Delay:       duration(time.Second * 1),
				URL:         "http://localhost:10101/index/equipment/query",
				Query:       "TopN(model, n=5)",
			},
		},
	}
}

type Tasks []Task

type Task struct {
	Connections int
	Delay       duration
	Query       string
	URL         string
}

type Config struct {
	Tasks Tasks
}

func (c *Config) Load(path string) error {
	_, err := toml.DecodeFile(path, c)
	return err
}

func (c *Config) Print(w io.Writer) error {
	return toml.NewEncoder(w).Encode(c)
}
