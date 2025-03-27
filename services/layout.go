package services

type Layout struct {
	Columns         int    `mapstructure:"columns" json:"columns"`
	BackgroundColor string `mapstructure:"background_color" json:"backgroundColor"`
	ButtonSize      int    `mapstructure:"button_size" json:"buttonSize"`
}
