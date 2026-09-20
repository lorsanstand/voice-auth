package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lorsanstand/voice-auth/internal/audio"
	"github.com/lorsanstand/voice-auth/internal/mathlib"
	"github.com/lorsanstand/voice-auth/internal/service"
)

const (
	defaultMaxUploadSize = 25 << 20
	frameSize            = 512
	hopSize              = 256
)

type Handler struct {
	biometry      *service.AudioBiometry
	ecapa         *service.ECAPABiometry
	maxUploadSize int64
}

func NewHandler(biometry *service.AudioBiometry, ecapa *service.ECAPABiometry) *Handler {
	return &Handler{
		biometry:      biometry,
		ecapa:         ecapa,
		maxUploadSize: defaultMaxUploadSize,
	}
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch {
	case request.URL.Path == "/" || request.URL.Path == "/index.html" || strings.HasPrefix(request.URL.Path, "/assets/"):
		h.serveWeb(writer, request)
	case request.URL.Path == "/voices" && request.Method == http.MethodGet:
		h.getAll(writer, request)
	case request.URL.Path == "/voices/register" && request.Method == http.MethodPost:
		h.register(writer, request)
	case request.URL.Path == "/voices/search" && request.Method == http.MethodPost:
		h.search(writer, request)
	case strings.HasPrefix(request.URL.Path, "/voices/") && request.Method == http.MethodDelete:
		h.delete(writer, request)
	case request.URL.Path == "/ecapa/voices" && request.Method == http.MethodGet:
		h.getAllECAPA(writer, request)
	case request.URL.Path == "/ecapa/voices/register" && request.Method == http.MethodPost:
		h.registerECAPA(writer, request)
	case request.URL.Path == "/ecapa/voices/search" && request.Method == http.MethodPost:
		h.searchECAPA(writer, request)
	case strings.HasPrefix(request.URL.Path, "/ecapa/voices/") && request.Method == http.MethodDelete:
		h.deleteECAPA(writer, request)
	default:
		writeError(writer, http.StatusNotFound, "route not found")
	}
}

func (h *Handler) getAll(writer http.ResponseWriter, request *http.Request) {
	voices, err := h.biometry.GetAllVoice(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(writer, http.StatusOK, voices)
}

func (h *Handler) register(writer http.ResponseWriter, request *http.Request) {
	file, name, err := h.readAudioForm(writer, request, true)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	frames, err := readFrames(file)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.biometry.RegisterVoice(request.Context(), frames, name); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrSmallSamples) {
			status = http.StatusBadRequest
		}
		writeError(writer, status, err.Error())
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]string{"message": "voice registered"})
}

func (h *Handler) search(writer http.ResponseWriter, request *http.Request) {
	file, _, err := h.readAudioForm(writer, request, false)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	frames, err := readFrames(file)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}

	voice, err := h.biometry.GetVoice(request.Context(), frames)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrSmallSamples) {
			status = http.StatusBadRequest
		}
		writeError(writer, status, err.Error())
		return
	}

	writeJSON(writer, http.StatusOK, voice)
}

func (h *Handler) delete(writer http.ResponseWriter, request *http.Request) {
	id := strings.TrimPrefix(request.URL.Path, "/voices/")
	if id == "" || strings.Contains(id, "/") {
		writeError(writer, http.StatusBadRequest, "voice id is required")
		return
	}

	if err := h.biometry.DeleteVoice(request.Context(), id); err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getAllECAPA(writer http.ResponseWriter, request *http.Request) {
	if h.ecapa == nil {
		writeError(writer, http.StatusServiceUnavailable, "ECAPA-TDNN model is not configured")
		return
	}

	voices, err := h.ecapa.GetAllVoice(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(writer, http.StatusOK, voices)
}

func (h *Handler) registerECAPA(writer http.ResponseWriter, request *http.Request) {
	if h.ecapa == nil {
		writeError(writer, http.StatusServiceUnavailable, "ECAPA-TDNN model is not configured")
		return
	}

	file, name, err := h.readAudioForm(writer, request, true)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	samples, err := readSamples(file)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.ecapa.RegisterVoice(request.Context(), samples, name); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrSmallSamples) {
			status = http.StatusBadRequest
		}
		writeError(writer, status, err.Error())
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]string{"message": "ECAPA voice registered"})
}

func (h *Handler) searchECAPA(writer http.ResponseWriter, request *http.Request) {
	if h.ecapa == nil {
		writeError(writer, http.StatusServiceUnavailable, "ECAPA-TDNN model is not configured")
		return
	}

	file, _, err := h.readAudioForm(writer, request, false)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	samples, err := readSamples(file)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}

	voice, err := h.ecapa.GetVoice(request.Context(), samples)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrSmallSamples) {
			status = http.StatusBadRequest
		}
		writeError(writer, status, err.Error())
		return
	}

	writeJSON(writer, http.StatusOK, voice)
}

func (h *Handler) deleteECAPA(writer http.ResponseWriter, request *http.Request) {
	if h.ecapa == nil {
		writeError(writer, http.StatusServiceUnavailable, "ECAPA-TDNN model is not configured")
		return
	}

	id := strings.TrimPrefix(request.URL.Path, "/ecapa/voices/")
	if id == "" || strings.Contains(id, "/") {
		writeError(writer, http.StatusBadRequest, "voice id is required")
		return
	}

	if err := h.ecapa.DeleteVoice(request.Context(), id); err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (h *Handler) readAudioForm(writer http.ResponseWriter, request *http.Request, requireName bool) (multipartFile, string, error) {
	request.Body = http.MaxBytesReader(writer, request.Body, h.maxUploadSize)
	if err := request.ParseMultipartForm(h.maxUploadSize); err != nil {
		return nil, "", fmt.Errorf("read multipart form: %w", err)
	}

	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, "", errors.New("form file 'file' is required")
	}

	name := strings.TrimSpace(request.FormValue("name"))
	if requireName && name == "" {
		file.Close()
		return nil, "", errors.New("form field 'name' is required")
	}

	return file, name, nil
}

type multipartFile interface {
	io.ReadSeeker
	io.Closer
}

func readFrames(file multipartFile) ([][]float64, error) {
	samples, err := readSamples(file)
	if err != nil {
		return nil, err
	}

	return mathlib.ExtractFrames(samples, frameSize, hopSize), nil
}

func readSamples(file multipartFile) ([]float64, error) {
	result, err := audio.ProcessWAVReader(file)
	if err != nil {
		return nil, fmt.Errorf("process WAV: %w", err)
	}

	return result.Samples, nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}
