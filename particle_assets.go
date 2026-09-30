package particleapp

import "embed"

// MediaFiles содержит только файлы-заглушки, которые при запуске копируются
// в MinIO. HTML-шаблонов и статических страниц в REST backend нет.
//
//go:embed resources/static/media/*
var MediaFiles embed.FS
