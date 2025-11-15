package initializers

import (
	"github.com/sivchari/govalid/internal/validator/registry"
	"github.com/sivchari/govalid/internal/validator/rules"
)

type DateInitializer struct{}

func (d DateInitializer) Marker() string { return "govalid:date" }

func (d DateInitializer) Init() registry.ValidatorFactory { return rules.ValidateDate }
