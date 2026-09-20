package main

import (
	"log"

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

	service.NewAudioBiometry(storage)

	//if len(os.Args) != 3 {
	//	fmt.Fprintf(os.Stderr, "usage: %s <file1.wav> <file2.wav>\n", os.Args[0])
	//	os.Exit(2)
	//}
	//
	//result, err := audio.ProcessWAV(os.Args[1])
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "read WAV: %v\n", err)
	//	os.Exit(1)
	//}
	//result1, err := audio.ProcessWAV(os.Args[2])
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "read WAV: %v\n", err)
	//	os.Exit(1)
	//}
	//
	//kus := mathlib.ExtractFrames(result.Samples, 512, 256)
	//kus1 := mathlib.ExtractFrames(result1.Samples, 512, 256)
	//
	//db, err := chromem.NewPersistentDB("./chromem_data", true)
	//if err != nil {
	//	log.Fatalf("Ошибка инициализации БД: %v", err)
	//}
	//
	//storage, err := inmemory.NewVectorStorage(db)
	//if err != nil {
	//	log.Fatalf("Ошибка инициализации voice collection: %v", err)
	//}
	//
	//serv := service.NewAudioBiometry(storage)
	//
	//v1 := serv.ToVector(kus)
	//v2 := serv.ToVector(kus1)
	//sim, err := CosineSimilarity(v1, v2)
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "compare vectors: %v\n", err)
	//	os.Exit(1)
	//}
	//
	//fmt.Printf("vector1=%v\n", v1)
	//fmt.Printf("vector2=%v\n", v2)
	//fmt.Printf("cosine_similarity=%.6f\n", sim)

}
