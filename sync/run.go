package sync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"sync2kindle/common"
	"sync2kindle/files"
	"sync2kindle/history"
	"sync2kindle/mtp"
	"sync2kindle/state"
	"sync2kindle/usb"
)

func RunUSB(ctx *cli.Context) error {
	return Sync(ctx, common.ProtocolUSB)
}

func RunMTP(ctx *cli.Context) error {
	return Sync(ctx, common.ProtocolMTP)
}

func Sync(ctx *cli.Context, protocol common.SupportedProtocols) error {
	env := ctx.Generic(state.FlagName).(*state.LocalEnv)
	log := env.Log.Named("sync")

	log.Info("Sync starting",
		zap.Stringer("protocol", protocol),
		zap.String("source", env.Cfg.SourcePath),
		zap.String("target", env.Cfg.TargetPath),
	)
	defer func(start time.Time) {
		log.Info("Sync finished",
			zap.Stringer("protocol", protocol),
			zap.Duration("elapsed", time.Since(start)))
	}(time.Now())

	// Source: local file system

	thumbDir, err := os.MkdirTemp("", "s2k-t-")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory: %w", err)
	}
	env.Cfg.Thumbnails.Dir = thumbDir
	env.Rpt.Store("thumbs", thumbDir)

	src, err := files.Connect(env.Cfg.SourcePath, "", &env.Cfg.Thumbnails, log)
	if err != nil {
		return fmt.Errorf("bad source path: %w", err)
	}
	defer src.Disconnect()

	// Target: device

	dev, err := connectDevice(ctx, protocol, env)
	if err != nil {
		return fmt.Errorf("unable to connect to device: %w", err)
	}
	defer dev.Disconnect()

	// History: local DB

	historyExists := true
	historyPath := filepath.Join(env.Cfg.HistoryPath, history.GetName(protocol, dev.UniqueID(), env.Cfg.TargetPath))
	_, err = os.Stat(historyPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("history database '%s' cannot be accessed: %w", historyPath, err)
		}
		historyExists = false
	}
	log.Debug("History database", zap.String("path", historyPath))

	if !historyExists {
		if err := history.Create(historyPath, log, protocol.String(), dev.UniqueID(), env.Cfg.TargetPath); err != nil {
			return fmt.Errorf("unable to create new history database '%s': %w", historyPath, err)
		}
	} else {
		env.Rpt.StoreCopy("history/original.db", historyPath)
	}

	hst, err := history.Connect(historyPath, log)
	if err != nil {
		return fmt.Errorf("history cannot be opened: %w", err)
	}
	defer func() {
		hst.Disconnect()
		env.Rpt.Store("history/updated.db", historyPath)
	}()
	log.Debug("History last step", zap.Int64("stepID", hst.StepID()))

	// See if anything needs to be done

	actions, localBooks, err := PrepareActions(src, dev, hst, env.Cfg, ctx.Bool("ignore-device-removals"), log)
	if err != nil {
		return fmt.Errorf("unable to prepare sync actions: %w", err)
	}
	if len(actions) == 0 {
		log.Info("Nothing to do")
	}

	// do the work

	dryRun := ctx.Bool("dry-run")
	for _, action := range actions {
		if err := action(dryRun, log); err != nil {
			return fmt.Errorf("action failed: %w", err)
		}
	}

	// Update history only if we had some actions or it is our first sync

	if !dryRun && (len(actions) != 0 || hst.StepID() == 0) {
		if err := hst.SaveObjectInfos(env.Cfg.SourcePath, env.Cfg.TargetPath, localBooks.SubsetByPath(env.Cfg.SourcePath)); err != nil {
			return fmt.Errorf("history objects cannot be saved: %w", err)
		}
		log.Debug("History next step", zap.Int64("stepID", hst.StepID()))
	}
	return nil
}

func connectDevice(ctx *cli.Context, protocol common.SupportedProtocols, env *state.LocalEnv) (driver, error) {
	switch protocol {
	case common.ProtocolUSB:
		return usb.Connect(
			strings.Join([]string{env.Cfg.TargetPath, common.ThumbnailFolder}, string(filepath.ListSeparator)),
			env.Cfg.DeviceSerial, ctx.Bool("unmount"), env.Log.Named("sync"))
	case common.ProtocolMTP:
		return mtp.Connect(
			strings.Join([]string{env.Cfg.TargetPath, common.ThumbnailFolder}, string(filepath.ListSeparator)),
			env.Cfg.DeviceSerial, env.Log.Named("sync"))
	default:
		return nil, fmt.Errorf("unsupported protocol requested for sync: %s", protocol)
	}
}
