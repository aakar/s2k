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
	"sync2kindle/config"
	"sync2kindle/files"
	"sync2kindle/history"
	"sync2kindle/mail"
	"sync2kindle/mtp"
	"sync2kindle/state"
	"sync2kindle/usbms"
)

func RunUSB(ctx *cli.Context) error {
	return Sync(ctx, common.ProtocolUSB)
}

func RunMTP(ctx *cli.Context) error {
	return Sync(ctx, common.ProtocolMTP)
}

func RunMail(ctx *cli.Context) error {
	return Sync(ctx, common.ProtocolMail)
}

func Sync(ctx *cli.Context, protocol common.SupportedProtocols) error {
	env := ctx.Generic(state.FlagName).(*state.LocalEnv)
	log := env.Log.Named("sync")

	if protocol == common.ProtocolMail {
		if !strings.Contains(env.Cfg.TargetPath, "@") {
			return fmt.Errorf("target is invalid e-mail address: %s", env.Cfg.TargetPath)
		}
		var supported, notSupported []string
		for _, ext := range env.Cfg.BookExtensions {
			if !common.IsSupportedEMailFormat(ext) {
				notSupported = append(notSupported, ext)
			} else {
				supported = append(supported, ext)
			}
		}
		if len(notSupported) > 0 {
			log.Warn("extensions not supported by e-mail are specified in configuration", zap.Strings("extensions", notSupported))
		}
		if len(supported) == 0 {
			return fmt.Errorf("no supported e-mail formats are specified in configuration")
		}
		env.Cfg.BookExtensions = supported
	}

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

	// do not look at thumbnails if e-mail delivery is requested
	var thumbsCfg *config.ThumbnailsConfig
	if protocol != common.ProtocolMail {
		thumbDir, err := os.MkdirTemp("", "s2k-t-")
		if err != nil {
			return fmt.Errorf("unable to create temporary directory: %w", err)
		}
		env.Cfg.Thumbnails.Dir = thumbDir
		env.Rpt.Store("thumbs", thumbDir)
		// indicate that thumbs need to be processed
		thumbsCfg = &env.Cfg.Thumbnails
	}

	src, err := files.Connect(env.Cfg.SourcePath, "", thumbsCfg, log)
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

	actions, localBooks, err := PrepareActions(src, dev, hst, env.Cfg, ctx.Bool("ignore-device-removals"), protocol == common.ProtocolMail, log)
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
		return usbms.Connect(
			strings.Join([]string{env.Cfg.TargetPath, common.ThumbnailFolder}, string(filepath.ListSeparator)),
			env.Cfg.DeviceSerial, ctx.Bool("unmount") && !ctx.Bool("dry-run"), env.Log.Named("sync"))
	case common.ProtocolMTP:
		return mtp.Connect(
			strings.Join([]string{env.Cfg.TargetPath, common.ThumbnailFolder}, string(filepath.ListSeparator)),
			env.Cfg.DeviceSerial, ctx.Bool("debug"), env.Log.Named("sync"))
	case common.ProtocolMail:
		debug := ctx.Bool("debug")
		if debug {
			mailDir, err := os.MkdirTemp("", "s2k-m-")
			if err != nil {
				return nil, fmt.Errorf("unable to create temporary directory: %w", err)
			}
			env.Cfg.Smtp.Dir = mailDir
			env.Rpt.Store("mails", mailDir)
		}
		return mail.Connect(env.Cfg.TargetPath, &env.Cfg.Smtp, debug, env.Log.Named("sync"))
	default:
		return nil, fmt.Errorf("unsupported protocol requested for sync: %s", protocol)
	}
}
