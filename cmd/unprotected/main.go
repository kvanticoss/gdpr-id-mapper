package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"github.com/kvanticoss/gdpr-id-mapper/internal/restapp"
	"github.com/kvanticoss/gdpr-id-mapper/internal/zapbadger"
	"github.com/kvanticoss/gdpr-id-mapper/pkg/idmapper"
	badgerAdaptor "github.com/kvanticoss/gdpr-id-mapper/pkg/kvstore/badger"
	"github.com/voi-go/svc"
	"go.uber.org/zap"
)

var (
	// Version the current code instance; usually commit hash
	Version = "SNAPSHOT"

	listenPort     int
	dbFolder       string
	defaultSalt    string
	defaultLiveTTL string
	truncate       bool
)

func loadFlags() {
	flag.IntVar(&listenPort, "port", 5000, "server listen address")
	flag.StringVar(&dbFolder, "db", "/tmp/id-mapper/", "Database folder")
	flag.StringVar(&defaultSalt, "default-salt", "", "Globally used salt")
	flag.BoolVar(&truncate, "truncate-db", false, "WARNING; will wipe existing databases if set to truthy values")
	flag.StringVar(&defaultLiveTTL, "ttl", "8760h", "Days for orphaned records to be keept")
	flag.Parse()
}

func main() {
	loadFlags()

	s, err := svc.New("gdpr-id-mapper", Version)
	svc.MustInit(s, err)

	logger := s.Logger().Named("DEVELOPMENT-UNPROTECTED-ID-MAPPER")

	opts := badger.DefaultOptions(dbFolder)
	opts.Truncate = truncate
	opts.SyncWrites = true
	opts.NumVersionsToKeep = 1
	opts.TableLoadingMode = options.LoadToRAM
	opts.ValueLogLoadingMode = options.FileIO
	opts.Logger = &zapbadger.LoggerBridge{
		SugaredLogger: logger.Named("badger-database").Sugar(),
	}
	db, err := badger.Open(opts)
	if err != nil {
		panic(err)
	}
	badger := badgerAdaptor.NewAdapter(db)
	defer badger.Close()

	ttl, err := time.ParseDuration(defaultLiveTTL)
	if err != nil {
		logger.Fatal(err.Error())
	}

	if defaultSalt == "" {
		defaultSalt = randomSalt()
		logger.Warn("NO DEFAULT SALT provided; using random salt (must be manually provided on next startup)", zap.String("default-salt", defaultSalt))
	}

	router := restapp.New(idmapper.NewGdprMapper(
		context.Background(),
		badger,
		[]byte(defaultSalt),
		ttl,
	))

	s.AddWorker("http-worker", NewHTTPWorker(Version,
		WithPort(listenPort),
		WithHandlerFunc(router.GetUnProctedMuxer().ServeHTTP),
	))

	s.Run()
}

func randomSalt() string {
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic(err) //If we are out of entropy; better crash then give false sense of working
	}

	return base64.URLEncoding.EncodeToString(randBytes)
}
