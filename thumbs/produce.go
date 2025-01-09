package thumbs

import (
	"path/filepath"
	"runtime/debug"

	"go.uber.org/zap"

	"sync2kindle/thumbs/mobi"
)

func ExtractThumbnail(fname string, params *ThumbnailsConfig, log *zap.Logger) (name string) {

	defer func() {
		// Sometimes device will have files we cannot recognize and parse
		if rec := recover(); rec != nil {
			log.Debug("Thumbnail processing ended with panic", zap.String("file", fname), zap.Any("record", rec), zap.ByteString("stack", debug.Stack()))
		}
	}()

	switch filepath.Ext(fname) {
	case ".mobi", ".azw3":
		r, err := mobi.NewReader(fname, params.Width, params.Height, params.Stretch, log)
		if err != nil {
			log.Warn("Thumbnail extraction failed", zap.String("file", fname), zap.Error(err))
		} else {
			name, err = r.SaveResult(params.Dir)
			if err != nil {
				log.Warn("Thumbnail saving failed", zap.String("file", fname), zap.Error(err))
			}
		}
	case ".kfx":
		// later
	case ".apnx":
		// ignore
	default:
		log.Debug("Unsupported file type for thumbnails", zap.String("file", fname))
	}
	return
}
