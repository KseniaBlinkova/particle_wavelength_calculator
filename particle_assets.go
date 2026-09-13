package particleapp

import "embed"

//go:embed resources/templates/*.html
//go:embed resources/static/css/*.css
var ParticleFiles embed.FS
