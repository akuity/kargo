package controller

// TODO: Document this
type RenderStrategy func(name, baseDir, envDir string) ([]byte, error)
