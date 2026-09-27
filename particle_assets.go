package particleapp

import "embed"

//go:embed resources/templates/*.html
//go:embed resources/static
var ParticleFiles embed.FS
