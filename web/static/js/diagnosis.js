// diagnosis.js - Crop image diagnosis

document.addEventListener('DOMContentLoaded', () => {
  const form = document.getElementById('diagnosis-form');
  if (!form) return;
  if (form.dataset.initialized === "true") return;
  form.dataset.initialized = "true";

  const dropZone = document.getElementById('image-drop-zone');
  const fileInput = document.getElementById('image-input');
  const preview = document.getElementById('image-preview');
  const uploadPrompt = document.getElementById('upload-prompt');
  const previewImg = document.getElementById('preview-img');
  const replaceBtn = document.getElementById('replace-image-btn');
  const removeBtn = document.getElementById('remove-image-btn');
  const submitBtn = document.getElementById('submit-btn');
  const progress = document.getElementById('upload-progress');
  const progressText = document.getElementById('progress-text');
  const imageError = document.getElementById('image-error');

  let selectedFile = null;
  let submitting = false;

  // Drag and drop
  dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('border-[#2E7D32]', 'bg-[#F5F7F5]');
  });

  dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('border-[#2E7D32]', 'bg-[#F5F7F5]');
  });

  dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('border-[#2E7D32]', 'bg-[#F5F7F5]');
    const files = e.dataTransfer.files;
    if (files.length > 0) handleFile(files[0]);
  });

  dropZone.addEventListener('click', () => fileInput.click());

  fileInput.addEventListener('change', () => {
    if (fileInput.files.length > 0) handleFile(fileInput.files[0]);
  });

  replaceBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    fileInput.click();
  });

  removeBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    removeFile();
  });

  function handleFile(file) {
    imageError.classList.add('hidden');

    const allowedTypes = ['image/jpeg', 'image/png', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      showImageError('Unsupported file type. Please upload JPEG, PNG, or WebP.');
      return;
    }

    const maxSize = 5 * 1024 * 1024;
    if (file.size > maxSize) {
      showImageError('Image too large. Maximum size is 5MB.');
      return;
    }

    selectedFile = file;
    const reader = new FileReader();
    reader.onload = (e) => {
      previewImg.src = e.target.result;
      uploadPrompt.classList.add('hidden');
      preview.classList.remove('hidden');
      dropZone.classList.remove('border-dashed');
      dropZone.classList.add('border-solid');
    };
    reader.readAsDataURL(file);
  }

  function removeFile() {
    selectedFile = null;
    fileInput.value = '';
    preview.classList.add('hidden');
    uploadPrompt.classList.remove('hidden');
    dropZone.classList.add('border-dashed');
    dropZone.classList.remove('border-solid');
    imageError.classList.add('hidden');
  }

  function showImageError(msg) {
    imageError.textContent = msg;
    imageError.classList.remove('hidden');
    selectedFile = null;
    fileInput.value = '';
  }

  // Form submission
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (submitting) return;

    const crop = document.getElementById('crop').value;
    const symptoms = document.getElementById('symptom_description').value.trim();

    if (!crop) {
      showToast('Please select a crop.', 'warning');
      return;
    }
    if (!symptoms) {
      showToast('Please describe the symptoms.', 'warning');
      return;
    }
    if (!selectedFile) {
      showToast('Please upload an image.', 'warning');
      return;
    }

    submitting = true;
    submitBtn.disabled = true;
    progress.classList.remove('hidden');
    progressText.textContent = 'Uploading image...';

    const formData = new FormData(form);
    formData.set('image', selectedFile);

    try {
      const res = await fetch('/api/v1/diagnoses', {
        method: 'POST',
        body: formData,
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Submission failed');
      }

      progressText.textContent = 'Analysis submitted. Redirecting...';
      const data = await res.json();
      window.location.href = '/diagnoses/' + data.id;
    } catch (e) {
      progress.classList.add('hidden');
      showToast(e.message || 'Failed to submit diagnosis. Please try again.', 'error');
      submitting = false;
      submitBtn.disabled = false;
    }
  });
});

function showToast(message, type) {
  const colors = {
    info: 'bg-blue-50 border-blue-200 text-blue-700',
    warning: 'bg-amber-50 border-amber-200 text-amber-700',
    error: 'bg-red-50 border-red-200 text-red-700',
    success: 'bg-green-50 border-green-200 text-green-700',
  };

  const toast = document.createElement('div');
  toast.className = 'fixed top-4 right-4 z-50 px-4 py-3 rounded-xl border text-sm shadow-lg max-w-sm ' + (colors[type] || colors.info);
  toast.textContent = message;
  document.body.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}
