package handlers

import (
	"html/template"
	"math/rand/v2"
)

// affirmations are the celebratory messages shown when all checks pass.
var affirmations = []string{
	"All clear!",
	"Ship it!",
	"Nailed it!",
	"All green!",
	"Smooth sailing!",
	"Looking good!",
	"Rock solid!",
	"On point!",
	"Crushing it!",
	"Zero issues!",
	"Clean sweep!",
	"Top notch!",
	"Flawless!",
	"All systems go!",
	"Nice work!",
	"Spot on!",
	"Well done!",
	"No worries!",
	"Locked in!",
	"All good!",
}

// TemplateFuncs returns the shared FuncMap used by all templates.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b int) int { return a * b },
		"affirm": func() string {
			return affirmations[rand.IntN(len(affirmations))]
		},
	}
}
