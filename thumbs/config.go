package thumbs

type ThumbnailsConfig struct {
	Width  int `yaml:"width" validate:"required,gt=0"`
	Height int `yaml:"height" validate:"required,gt=0"`

	Dir string `yaml:"-"` // internal use only
}
