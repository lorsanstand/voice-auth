package main

import (
	"log"
	"net/http"
	"os"

	"github.com/lorsanstand/voice-auth/internal/embeddedmodel"
	"github.com/lorsanstand/voice-auth/internal/httpapi"
	"github.com/lorsanstand/voice-auth/internal/service"
	inmemory "github.com/lorsanstand/voice-auth/internal/storage/in-memory"
	"github.com/philippgille/chromem-go"
)

func main() {
	db, err := chromem.NewPersistentDB("./chromem_data", true)
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	storage, err := inmemory.NewVectorStorage(db)
	if err != nil {
		log.Fatalf("Ошибка инициализации voice collection: %v", err)
	}
	if err := storage.SetEmbeddingSize(48); err != nil {
		log.Fatalf("Ошибка настройки размера MFCC-вектора: %v", err)
	}

	biometry := service.NewAudioBiometry(storage)
	ecapaStorage, err := inmemory.NewVectorStorageWithCollection(db, "voice_ecapa")
	if err != nil {
		log.Fatalf("Ошибка инициализации ECAPA voice collection: %v", err)
	}

	modelPath := os.Getenv("ECAPA_MODEL_PATH")
	cleanupModel := func() {}
	if modelPath == "" {
		modelPath, cleanupModel, err = embeddedmodel.Materialize()
		if err != nil {
			log.Printf("Не удалось извлечь встроенную модель: %v", err)
		}
	}
	defer cleanupModel()

	ecapa, err := service.NewECAPABiometry(ecapaStorage, modelPath)
	if err != nil {
		log.Printf("ECAPA-TDNN отключён: %v", err)
	}
	defer ecapa.Close()

	handler := httpapi.NewHandler(biometry, ecapa)

	log.Println("HTTP server listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
