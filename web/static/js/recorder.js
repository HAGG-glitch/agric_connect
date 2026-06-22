// recorder.js - Voice recording and transcription

(function() {
  'use strict';

  let mediaRecorder = null;
  let audioChunks = [];
  let stream = null;
  let timerInterval = null;
  let startTime = null;
  let isRecording = false;

  const MAX_RECORDING_SECONDS = 60;

  const RecorderUI = {
    container: null,
    micBtn: null,
    recordingIndicator: null,
    timer: null,
    stopBtn: null,
    cancelBtn: null,
    playback: null,
    transcribeBtn: null,
    transcriptField: null,
    useBtn: null,
    retryBtn: null,
    krioNotice: null,
  };

  function init(config) {
    if (!config || !config.containerId) return;

    const container = document.getElementById(config.containerId);
    if (!container) return;

    RecorderUI.container = container;

    container.innerHTML = `
      <div class="flex items-center gap-2">
        <button id="mic-btn" type="button" class="p-2 text-[#6B7280] hover:text-[#2E7D32] hover:bg-[#F5F7F5] rounded-lg transition-colors" title="Record voice">
          <i data-lucide="mic" class="w-5 h-5"></i>
        </button>
      </div>
      <div id="recorder-panel" class="hidden space-y-3 mt-2 p-3 bg-[#F5F7F5] rounded-xl">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <div id="recording-dot" class="w-3 h-3 bg-red-500 rounded-full hidden animate-pulse"></div>
            <span id="recording-timer" class="text-sm font-mono text-[#6B7280]">0:00</span>
          </div>
          <div class="flex gap-1">
            <button id="stop-recording-btn" class="hidden p-1.5 bg-red-100 text-red-600 rounded-lg hover:bg-red-200 transition-colors" title="Stop recording">
              <i data-lucide="square" class="w-4 h-4"></i>
            </button>
            <button id="cancel-recording-btn" class="hidden p-1.5 text-[#6B7280] hover:bg-[#E5E7EB] rounded-lg transition-colors" title="Cancel recording">
              <i data-lucide="x" class="w-4 h-4"></i>
            </button>
          </div>
        </div>
        <audio id="recording-playback" class="hidden w-full" controls></audio>
        <div class="flex gap-2">
          <button id="transcribe-btn" class="hidden flex-1 bg-[#2E7D32] text-white rounded-lg py-1.5 text-sm font-medium hover:bg-[#245F27] transition-colors disabled:opacity-40">
            Transcribe
          </button>
          <button id="retry-recording-btn" class="hidden px-3 py-1.5 border border-[#E5E7EB] rounded-lg text-sm text-[#6B7280] hover:bg-white transition-colors">
            Re-record
          </button>
        </div>
        <div id="krio-transcription-notice" class="hidden text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-2">
          <i data-lucide="alert-triangle" class="w-3 h-3 inline mr-1"></i>
          Krio voice transcription is experimental. Please review and correct the transcript before sending it.
        </div>
        <textarea id="transcript-text" class="hidden w-full border border-[#E5E7EB] rounded-lg px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-[#2E7D32]/30" rows="2" placeholder="Transcript will appear here..."></textarea>
        <button id="use-transcript-btn" class="hidden bg-[#2E7D32] text-white rounded-lg py-1.5 text-sm font-medium hover:bg-[#245F27] transition-colors">
          Use Transcript
        </button>
      </div>
    `;

    RecorderUI.micBtn = document.getElementById('mic-btn');
    RecorderUI.panel = document.getElementById('recorder-panel');
    RecorderUI.recordingDot = document.getElementById('recording-dot');
    RecorderUI.timer = document.getElementById('recording-timer');
    RecorderUI.stopBtn = document.getElementById('stop-recording-btn');
    RecorderUI.cancelBtn = document.getElementById('cancel-recording-btn');
    RecorderUI.playback = document.getElementById('recording-playback');
    RecorderUI.transcribeBtn = document.getElementById('transcribe-btn');
    RecorderUI.transcriptField = document.getElementById('transcript-text');
    RecorderUI.useBtn = document.getElementById('use-transcript-btn');
    RecorderUI.retryBtn = document.getElementById('retry-recording-btn');
    RecorderUI.krioNotice = document.getElementById('krio-transcription-notice');

    if (typeof lucide !== 'undefined') lucide.createIcons();

    // Check browser support
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      RecorderUI.micBtn.title = 'Voice recording not supported in this browser';
      RecorderUI.micBtn.classList.add('opacity-40', 'cursor-not-allowed');
      return;
    }

    RecorderUI.micBtn.addEventListener('click', startRecording);
    RecorderUI.stopBtn.addEventListener('click', stopRecording);
    RecorderUI.cancelBtn.addEventListener('click', cancelRecording);
    RecorderUI.transcribeBtn.addEventListener('click', () => transcribeAudio(config));
    RecorderUI.retryBtn.addEventListener('click', resetRecorder);
    RecorderUI.useBtn.addEventListener('click', () => useTranscript(config));

    // Cleanup on page navigation
    window.addEventListener('pagehide', function() {
      if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
      }
      if (stream) {
        stream.getTracks().forEach(function(t) { t.stop(); });
        stream = null;
      }
      if (mediaRecorder && mediaRecorder.state !== 'inactive') {
        mediaRecorder.stop();
      }
    });
  }

  async function startRecording() {
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true });

      const mimeType = getSupportedMimeType();
      audioChunks = [];

      mediaRecorder = new MediaRecorder(stream, mimeType ? { mimeType } : {});

      mediaRecorder.ondataavailable = (e) => {
        if (e.data.size > 0) audioChunks.push(e.data);
      };

      mediaRecorder.onstop = () => {
        const blob = new Blob(audioChunks, { type: mediaRecorder.mimeType });
        showPlayback(blob);
        releaseMicrophone();
      };

      mediaRecorder.start(100);
      isRecording = true;

      RecorderUI.micBtn.classList.add('hidden');
      RecorderUI.panel.classList.remove('hidden');
      RecorderUI.recordingDot.classList.remove('hidden');
      RecorderUI.stopBtn.classList.remove('hidden');
      RecorderUI.cancelBtn.classList.remove('hidden');

      startTime = Date.now();
      updateTimer();
      timerInterval = setInterval(updateTimer, 1000);

    } catch (err) {
      if (err.name === 'NotAllowedError') {
        showError('Microphone permission denied. Please allow microphone access.');
      } else if (err.name === 'NotFoundError') {
        showError('No microphone found.');
      } else {
        showError('Could not access microphone: ' + err.message);
      }
    }
  }

  function stopRecording() {
    if (mediaRecorder && mediaRecorder.state !== 'inactive') {
      mediaRecorder.stop();
      isRecording = false;
      clearInterval(timerInterval);
      RecorderUI.recordingDot.classList.add('hidden');
      RecorderUI.stopBtn.classList.add('hidden');
      RecorderUI.cancelBtn.classList.add('hidden');
    }
  }

  function cancelRecording() {
    if (mediaRecorder && mediaRecorder.state !== 'inactive') {
      mediaRecorder.onstop = null;
      mediaRecorder.stop();
    }
    isRecording = false;
    clearInterval(timerInterval);
    releaseMicrophone();
    resetRecorder();
  }

  function releaseMicrophone() {
    if (stream) {
      stream.getTracks().forEach(t => t.stop());
      stream = null;
    }
  }

  function showPlayback(blob) {
    const url = URL.createObjectURL(blob);
    RecorderUI.playback.src = url;
    RecorderUI.playback.classList.remove('hidden');
    RecorderUI.transcribeBtn.classList.remove('hidden');
    RecorderUI.retryBtn.classList.remove('hidden');
    RecorderUI.transcribeBtn.disabled = false;
    RecorderUI.blob = blob;
  }

  async function transcribeAudio(config) {
    if (!RecorderUI.blob) return;

    RecorderUI.transcribeBtn.disabled = true;
    RecorderUI.transcribeBtn.textContent = 'Transcribing...';

    const formData = new FormData();
    formData.append('audio', RecorderUI.blob, 'recording.' + getExtension(RecorderUI.blob.type));
    var currentLang = typeof State !== 'undefined' && State.language ? State.language : (config.language || 'auto');
    formData.append('language_hint', currentLang);

    console.log('recorder: transcription request starting', {
      blobType: RecorderUI.blob.type,
      blobSize: RecorderUI.blob.size,
    });

    try {
      const res = await fetch('/api/v1/ai/transcribe', {
        method: 'POST',
        credentials: 'same-origin',
        body: formData,
      });

      console.log('recorder: transcription response', { status: res.status });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Transcription failed');
      }

      const data = await res.json();
      RecorderUI.transcriptField.value = data.transcript;
      RecorderUI.transcriptField.classList.remove('hidden');
      RecorderUI.transcribeBtn.classList.add('hidden');
      RecorderUI.retryBtn.classList.add('hidden');
      RecorderUI.useBtn.classList.remove('hidden');
      RecorderUI.playback.classList.add('hidden');

      if (data.requires_confirmation) {
        RecorderUI.krioNotice.classList.remove('hidden');
        if (typeof lucide !== 'undefined') lucide.createIcons();
      }

    } catch (e) {
      showError(e.message || 'Transcription failed. Please try again.');
      RecorderUI.transcribeBtn.disabled = false;
      RecorderUI.transcribeBtn.textContent = 'Transcribe';
    }
  }

  function useTranscript(config) {
    const text = RecorderUI.transcriptField.value.trim();
    if (!text) return;

    const input = document.getElementById('message-input');
    if (input) {
      input.value = text;
      input.focus();
      autoResizeTextarea(input);
    }

    resetRecorder();
  }

  function resetRecorder() {
    RecorderUI.panel.classList.add('hidden');
    RecorderUI.micBtn.classList.remove('hidden');
    RecorderUI.transcribeBtn.classList.remove('hidden');
    RecorderUI.transcribeBtn.textContent = 'Transcribe';
    RecorderUI.transcribeBtn.disabled = false;
    RecorderUI.transcriptField.classList.add('hidden');
    RecorderUI.transcriptField.value = '';
    RecorderUI.useBtn.classList.add('hidden');
    RecorderUI.retryBtn.classList.add('hidden');
    RecorderUI.playback.classList.add('hidden');
    RecorderUI.playback.src = '';
    RecorderUI.krioNotice.classList.add('hidden');
    RecorderUI.timer.textContent = '0:00';
    RecorderUI.blob = null;

    if (RecorderUI.playback.src) {
      URL.revokeObjectURL(RecorderUI.playback.src);
    }
  }

  function updateTimer() {
    const elapsed = Math.floor((Date.now() - startTime) / 1000);
    if (elapsed >= MAX_RECORDING_SECONDS) {
      stopRecording();
      return;
    }
    const mins = Math.floor(elapsed / 60);
    const secs = elapsed % 60;
    RecorderUI.timer.textContent = mins + ':' + (secs < 10 ? '0' : '') + secs;
  }

  function getSupportedMimeType() {
    const types = [
      'audio/webm;codecs=opus',
      'audio/webm',
      'audio/ogg;codecs=opus',
      'audio/mp4',
      'audio/wav',
    ];
    for (const type of types) {
      if (MediaRecorder.isTypeSupported(type)) return type;
    }
    return null;
  }

  function getExtension(mimeType) {
    const map = {
      'audio/webm': 'webm',
      'audio/ogg': 'ogg',
      'audio/mp4': 'mp4',
      'audio/wav': 'wav',
      'audio/mpeg': 'mp3',
    };
    return map[mimeType] || 'webm';
  }

  function showError(msg) {
    const toast = document.createElement('div');
    toast.className = 'fixed top-4 right-4 z-50 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-xl text-sm shadow-lg max-w-sm';
    toast.textContent = msg;
    document.body.appendChild(toast);
    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.3s';
      setTimeout(() => toast.remove(), 300);
    }, 4000);
  }

  window.Recorder = { init };
})();
