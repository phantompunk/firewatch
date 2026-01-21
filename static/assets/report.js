/**
 * Firewatch Reports - Form Handler
 * Handles file uploads, drag-and-drop, and basic client-side validation.
 * No tracking, no analytics, no external dependencies.
 */

(function() {
    'use strict';

    // DOM Elements
    const form = document.getElementById('report-form');
    const fileInput = document.getElementById('media');
    const fileUploadArea = document.getElementById('file-upload-area');
    const fileList = document.getElementById('file-list');
    const submitBtn = document.getElementById('submit-btn');

    // State
    let selectedFiles = [];
    const MAX_FILES = 5;
    const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

    // Initialize
    function init() {
        if (!form) return;

        setupFileUpload();
        setupFormSubmission();
        setupClearWarning();
    }

    // File Upload Handling
    function setupFileUpload() {
        if (!fileInput || !fileUploadArea) return;

        // File input change
        fileInput.addEventListener('change', handleFileSelect);

        // Drag and drop
        fileUploadArea.addEventListener('dragover', function(e) {
            e.preventDefault();
            e.stopPropagation();
            this.classList.add('dragover');
        });

        fileUploadArea.addEventListener('dragleave', function(e) {
            e.preventDefault();
            e.stopPropagation();
            this.classList.remove('dragover');
        });

        fileUploadArea.addEventListener('drop', function(e) {
            e.preventDefault();
            e.stopPropagation();
            this.classList.remove('dragover');

            const files = e.dataTransfer.files;
            handleFiles(files);
        });
    }

    function handleFileSelect(e) {
        handleFiles(e.target.files);
    }

    function handleFiles(files) {
        const allowedTypes = [
            'image/jpeg',
            'image/png',
            'image/gif',
            'image/webp',
            'video/mp4',
            'video/webm'
        ];

        for (let i = 0; i < files.length; i++) {
            const file = files[i];

            // Check max files
            if (selectedFiles.length >= MAX_FILES) {
                alert('Maximum ' + MAX_FILES + ' files allowed.');
                break;
            }

            // Check file type
            if (!allowedTypes.includes(file.type)) {
                alert('File type not supported: ' + file.name);
                continue;
            }

            // Check file size
            if (file.size > MAX_FILE_SIZE) {
                alert('File too large (max 10MB): ' + file.name);
                continue;
            }

            // Check for duplicates
            const isDuplicate = selectedFiles.some(function(f) {
                return f.name === file.name && f.size === file.size;
            });
            if (isDuplicate) continue;

            selectedFiles.push(file);
            renderFileList();
        }

        // Clear the input so the same file can be selected again if removed
        fileInput.value = '';
    }

    function renderFileList() {
        if (!fileList) return;

        fileList.innerHTML = '';

        selectedFiles.forEach(function(file, index) {
            const item = document.createElement('div');
            item.className = 'file-item';

            const nameSpan = document.createElement('span');
            nameSpan.className = 'filename';
            nameSpan.textContent = file.name + ' (' + formatFileSize(file.size) + ')';

            const statusSpan = document.createElement('span');
            statusSpan.className = 'status';
            statusSpan.textContent = 'Ready';

            const removeBtn = document.createElement('button');
            removeBtn.type = 'button';
            removeBtn.className = 'remove';
            removeBtn.textContent = '×';
            removeBtn.setAttribute('aria-label', 'Remove ' + file.name);
            removeBtn.addEventListener('click', function() {
                removeFile(index);
            });

            item.appendChild(nameSpan);
            item.appendChild(statusSpan);
            item.appendChild(removeBtn);
            fileList.appendChild(item);
        });
    }

    function removeFile(index) {
        selectedFiles.splice(index, 1);
        renderFileList();
    }

    function formatFileSize(bytes) {
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    }

    // Form Submission
    function setupFormSubmission() {
        form.addEventListener('submit', function(e) {
            e.preventDefault();

            // Basic validation
            const description = form.querySelector('#description');
            if (!description.value.trim()) {
                alert('Please provide a description of the activity.');
                description.focus();
                return;
            }

            // Disable submit button and show loading
            submitBtn.disabled = true;
            submitBtn.classList.add('loading');
            submitBtn.textContent = 'Submitting...';

            // Build FormData with selected files
            const formData = new FormData(form);

            // Remove the original file input data and add our tracked files
            formData.delete('media');
            selectedFiles.forEach(function(file) {
                formData.append('media', file);
            });

            // Submit via fetch for better UX
            fetch('/api/submit', {
                method: 'POST',
                body: formData
            })
            .then(function(response) {
                if (response.redirected) {
                    // Follow the redirect to success page
                    window.location.href = response.url;
                } else if (response.ok) {
                    window.location.href = '/submitted.html';
                } else {
                    return response.text().then(function(text) {
                        throw new Error(text || 'Submission failed');
                    });
                }
            })
            .catch(function(error) {
                alert('Error submitting report: ' + error.message + '\n\nPlease try again.');
                submitBtn.disabled = false;
                submitBtn.classList.remove('loading');
                submitBtn.textContent = 'Submit Report';
            });
        });
    }

    // Clear warning on navigation
    function setupClearWarning() {
        let hasUnsavedData = false;

        // Track if user has entered data
        form.addEventListener('input', function() {
            hasUnsavedData = true;
        });

        // Warn before leaving if there's unsaved data
        window.addEventListener('beforeunload', function(e) {
            if (hasUnsavedData) {
                e.preventDefault();
                e.returnValue = '';
            }
        });

        // Clear the warning when form is submitted or reset
        form.addEventListener('submit', function() {
            hasUnsavedData = false;
        });

        form.addEventListener('reset', function() {
            hasUnsavedData = false;
            selectedFiles = [];
            renderFileList();
        });
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
