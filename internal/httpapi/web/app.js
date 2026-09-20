const recordedFiles = new WeakMap();
const recorderResetters = new WeakMap();

function showToast(message, isError = false) {
  const toast = document.querySelector("#toast");
  toast.textContent = message;
  toast.className = `toast show${isError ? " error" : ""}`;
  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => {
    toast.className = "toast";
  }, 3500);
}

function setLoading(button, loading) {
  button.disabled = loading;
  button.dataset.label ??= button.textContent;
  button.textContent = loading ? "Обработка…" : button.dataset.label;
}

async function requestJSON(url, options = {}) {
  const response = await fetch(url, options);
  const text = await response.text();
  let data = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      throw new Error("Сервер вернул некорректный ответ");
    }
  }
  if (!response.ok) {
    throw new Error(data.error || `Ошибка запроса: ${response.status}`);
  }
  return data;
}

function connectFileInput(input) {
  const name = input.closest(".file-drop").querySelector("[data-file-name]");
  input.addEventListener("change", () => {
    recordedFiles.delete(input);
    name.textContent = input.files[0]?.name || "Файл ещё не выбран";
  });
}

function buildAudioFormData(form) {
  const input = form.querySelector('input[type="file"]');
  const file = recordedFiles.get(input) || input.files[0];
  if (!file) {
    throw new Error("Выбери WAV-файл или сначала запиши голос");
  }

  const formData = new FormData();
  form.querySelectorAll("[name]").forEach((field) => {
    if (field.type !== "file") {
      formData.append(field.name, field.value);
    }
  });
  formData.append(input.name, file, file.name);
  return formData;
}

function supportedRecordingType() {
  if (!window.MediaRecorder || typeof MediaRecorder.isTypeSupported !== "function") {
    return "";
  }

  return [
    "audio/webm;codecs=opus",
    "audio/webm",
    "audio/ogg;codecs=opus",
    "audio/ogg",
  ].find((type) => MediaRecorder.isTypeSupported(type)) || "";
}

function writeASCII(view, offset, value) {
  for (let index = 0; index < value.length; index += 1) {
    view.setUint8(offset + index, value.charCodeAt(index));
  }
}

function createWavFile(audioBuffer, filename) {
  const channelCount = audioBuffer.numberOfChannels;
  const sampleCount = audioBuffer.length;
  const sampleRate = audioBuffer.sampleRate;
  const bytesPerSample = 2;
  const dataSize = sampleCount * bytesPerSample;
  const buffer = new ArrayBuffer(44 + dataSize);
  const view = new DataView(buffer);

  writeASCII(view, 0, "RIFF");
  view.setUint32(4, 36 + dataSize, true);
  writeASCII(view, 8, "WAVE");
  writeASCII(view, 12, "fmt ");
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * bytesPerSample, true);
  view.setUint16(32, bytesPerSample, true);
  view.setUint16(34, 16, true);
  writeASCII(view, 36, "data");
  view.setUint32(40, dataSize, true);

  const channels = Array.from({ length: channelCount }, (_, index) =>
    audioBuffer.getChannelData(index),
  );
  for (let sample = 0; sample < sampleCount; sample += 1) {
    let mixed = 0;
    for (const channel of channels) {
      mixed += channel[sample];
    }
    mixed /= channelCount;
    mixed = Math.max(-1, Math.min(1, mixed));
    const value = mixed < 0 ? mixed * 0x8000 : mixed * 0x7fff;
    view.setInt16(44 + sample * bytesPerSample, value, true);
  }

  return new File([buffer], filename, { type: "audio/wav" });
}

async function recordingToWav(blob) {
  const AudioContext = window.AudioContext || window.webkitAudioContext;
  if (!AudioContext) {
    throw new Error("Браузер не умеет преобразовывать запись в WAV");
  }

  const context = new AudioContext();
  try {
    const audioBuffer = await context.decodeAudioData(await blob.arrayBuffer());
    return createWavFile(audioBuffer, `voice-${Date.now()}.wav`);
  } finally {
    await context.close();
  }
}

function connectRecorder(container) {
  const input = document.querySelector(`#${container.dataset.input}`);
  const startButton = container.querySelector("[data-record-start]");
  const stopButton = container.querySelector("[data-record-stop]");
  const status = container.querySelector("[data-record-status]");
  const recordingSupported = Boolean(
    navigator.mediaDevices?.getUserMedia && window.MediaRecorder,
  );
  let recorder;
  let stream;
  let chunks = [];
  let timer;
  let startedAt;

  function reset() {
    status.textContent = "Можно записать голос через микрофон";
    startButton.disabled = !recordingSupported;
    stopButton.disabled = true;
  }

  recorderResetters.set(input, reset);

  if (!recordingSupported) {
    startButton.disabled = true;
    status.textContent = "Запись не поддерживается этим браузером";
    return;
  }

  startButton.addEventListener("click", async () => {
    startButton.disabled = true;
    status.textContent = "Запрашиваем доступ к микрофону…";
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mimeType = supportedRecordingType();
      recorder = mimeType
        ? new MediaRecorder(stream, { mimeType })
        : new MediaRecorder(stream);
      chunks = [];
      startedAt = Date.now();
      recorder.addEventListener("dataavailable", (event) => {
        if (event.data.size > 0) chunks.push(event.data);
      });
      recorder.addEventListener("stop", async () => {
        window.clearInterval(timer);
        stream.getTracks().forEach((track) => track.stop());
        stopButton.disabled = true;
        status.textContent = "Подготавливаем WAV-файл…";
        try {
          const blob = new Blob(chunks, { type: recorder.mimeType });
          const file = await recordingToWav(blob);
          recordedFiles.set(input, file);
          input.value = "";
          input.closest("label").querySelector("[data-file-name]").textContent = file.name;
          status.textContent = "Запись готова к отправке";
          startButton.disabled = false;
        } catch (error) {
          status.textContent = error.message;
          startButton.disabled = false;
        }
      }, { once: true });
      recorder.start();
      stopButton.disabled = false;
      status.textContent = "Идёт запись… 00:00";
      timer = window.setInterval(() => {
        const seconds = Math.floor((Date.now() - startedAt) / 1000);
        status.textContent = `Идёт запись… ${String(Math.floor(seconds / 60)).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`;
      }, 1000);
    } catch (error) {
      stream?.getTracks().forEach((track) => track.stop());
      startButton.disabled = false;
      status.textContent = error.name === "NotAllowedError"
        ? "Доступ к микрофону запрещён"
        : "Не удалось начать запись";
    }
  });

  stopButton.addEventListener("click", () => {
    if (recorder?.state === "recording") {
      recorder.stop();
      status.textContent = "Останавливаем запись…";
    }
  });
}

function renderResult(resultElement, voice) {
  resultElement.className = "result";
  resultElement.innerHTML = `
    <div class="result-name">${escapeHTML(voice.Content || "Без имени")}</div>
    <div class="score">Сходство: ${(Number(voice.Similarity) * 100).toFixed(2)}%</div>
  `;
}

async function loadVoices(engine) {
  engine.voicesList.innerHTML = '<div class="loading">Загрузка профилей…</div>';
  try {
    const voices = await requestJSON(`${engine.basePath}/voices`);
    if (!voices.length) {
      engine.voicesList.innerHTML = '<div class="empty-list">Пока нет зарегистрированных голосов.</div>';
      return;
    }
    engine.voicesList.innerHTML = voices.map((voice) => `
      <div class="voice-row">
        <div class="voice-info">
          <div class="voice-name">${escapeHTML(voice.Content || "Без имени")}</div>
          <div class="voice-id">${escapeHTML(voice.ID)}</div>
        </div>
        <button class="delete-button" type="button" data-id="${escapeHTML(voice.ID)}">Удалить</button>
      </div>
    `).join("");
    engine.voicesList.querySelectorAll(".delete-button").forEach((button) => {
      button.addEventListener("click", () => deleteVoice(engine, button.dataset.id));
    });
  } catch (error) {
    engine.voicesList.innerHTML = `<div class="empty-list">${escapeHTML(error.message)}</div>`;
  }
}

async function deleteVoice(engine, id) {
  if (!window.confirm("Удалить этот голос из хранилища?")) return;
  try {
    await requestJSON(`${engine.basePath}/voices/${encodeURIComponent(id)}`, { method: "DELETE" });
    showToast("Голос удалён");
    await loadVoices(engine);
  } catch (error) {
    showToast(error.message, true);
  }
}

function setupEngine(engine) {
  engine.registerForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const button = engine.registerForm.querySelector('button[type="submit"]');
    setLoading(button, true);
    try {
      await requestJSON(`${engine.basePath}/voices/register`, {
        method: "POST",
        body: buildAudioFormData(engine.registerForm),
      });
      engine.registerForm.reset();
      const input = engine.registerForm.querySelector('input[type="file"]');
      recordedFiles.delete(input);
      recorderResetters.get(input)?.();
      engine.registerForm.querySelector("[data-file-name]").textContent = "Файл ещё не выбран";
      showToast(`${engine.name}: голос успешно зарегистрирован`);
      await loadVoices(engine);
    } catch (error) {
      showToast(error.message, true);
    } finally {
      setLoading(button, false);
    }
  });

  engine.searchForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const button = engine.searchForm.querySelector('button[type="submit"]');
    setLoading(button, true);
    engine.searchResult.className = "result empty";
    engine.searchResult.textContent = "Ищем ближайший голос…";
    try {
      const voice = await requestJSON(`${engine.basePath}/voices/search`, {
        method: "POST",
        body: buildAudioFormData(engine.searchForm),
      });
      renderResult(engine.searchResult, voice);
    } catch (error) {
      engine.searchResult.className = "result empty";
      engine.searchResult.textContent = error.message;
    } finally {
      setLoading(button, false);
    }
  });

  engine.refreshButton.addEventListener("click", () => loadVoices(engine));
  loadVoices(engine);
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (character) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#039;",
  }[character]));
}

const engines = [
  {
    name: "MFCC",
    basePath: "",
    registerForm: document.querySelector("#register-form"),
    searchForm: document.querySelector("#search-form"),
    voicesList: document.querySelector("#voices-list"),
    searchResult: document.querySelector("#search-result"),
    refreshButton: document.querySelector("#refresh-button"),
  },
  {
    name: "ECAPA-TDNN",
    basePath: "/ecapa",
    registerForm: document.querySelector("#ecapa-register-form"),
    searchForm: document.querySelector("#ecapa-search-form"),
    voicesList: document.querySelector("#ecapa-voices-list"),
    searchResult: document.querySelector("#ecapa-search-result"),
    refreshButton: document.querySelector("#ecapa-refresh-button"),
  },
];

document.querySelectorAll('input[type="file"]').forEach(connectFileInput);
document.querySelectorAll("[data-recorder]").forEach(connectRecorder);
engines.forEach(setupEngine);
