(require 'go-mode)

(defun gogetdef (point)
  "Use the gogetdef tool to find the definition for an identifier at POINT.

You can install gogetdef with 'go get -u github.com/JohnWall2016/gogetdef'."
  (if (not (buffer-file-name (go--coverage-origin-buffer)))
      ;; TODO: gogetdef supports unsaved files, but not introducing
      ;; new artifical files, so this limitation will stay for now.
      (error "Cannot use gogetdef on a buffer without a file name"))
  (let ((buff (go--coverage-origin-buffer))
        (posn (if (eq system-type 'windows-nt)
                  (format "%s:#%d" (file-truename buffer-file-name) (1- (position-bytes point)))
                (format "%s:#%d" (shell-quote-argument (file-truename buffer-file-name)) (1- (position-bytes point)))))
        (out (godoc--get-buffer "<gogetdef>")))
  (with-current-buffer (get-buffer-create "*go-getdef-input*")
    (setq-local inhibit-eol-conversion t)
    (setq buffer-read-only nil)
    (erase-buffer)
    (go--insert-modified-files buff)
    (call-process-region (point-min) (point-max) "gogetdef" nil out nil
                         "-modified"
                         (format "-pos=%s" posn)))
  (prog1
      (with-current-buffer out
        (split-string (buffer-substring-no-properties (point-min) (point-max)) "\n" nil))
    (kill-buffer out))))

(defun gogetdef-jump (point &optional other-window)
  "Jump to the definition of the expression at POINT."
  (interactive "d")
  (condition-case nil
      (let* ((iod (gogetdef point))
             (sig (nth 0 iod))
             (pos (nth 1 iod)))
        (if (not (string= "gogetdef-return" sig))
            (message "%s" (mapconcat #'identity iod "\n"))
          (if (or (not pos) (string= pos ""))
              (message "No definition found for expression at point %S" iod)
            (push-mark)
            (if (eval-when-compile (fboundp 'xref-push-marker-stack))
                ;; TODO: Integrate this facility with XRef.
                (xref-push-marker-stack)
              (ring-insert find-tag-marker-ring (point-marker)))
            (godef--find-file-line-column pos other-window))))
    (file-error (message "Could not run gogetdef binary"))))

(defun gogetdef-describe (point)
  "Describe the expression at POINT."
  (interactive "d")
  (condition-case nil
      (let* ((iod (gogetdef point))
             (sig (car iod))
             (dcl (cddr iod)))
        (if (not (string= "gogetdef-return" sig))
            (message "%s" (mapconcat #'identity iod "\n"))
          (if (not dcl)
              (message "No definition found for expression at point %S" iod)
            (message "%s" (mapconcat #'identity dcl "\n")))))
    (file-error (message "Could not run gogetdef binary"))))

(defun gogetdef-all (point)
  "Use the gogetdef tool to find the definition for an identifier at POINT.

You can install gogetdef with 'go get -u github.com/JohnWall2016/gogetdef'."
  (if (not (buffer-file-name (go--coverage-origin-buffer)))
      ;; TODO: gogetdef supports unsaved files, but not introducing
      ;; new artifical files, so this limitation will stay for now.
      (error "Cannot use gogetdef on a buffer without a file name"))
  (let ((buff (go--coverage-origin-buffer))
        (posn (if (eq system-type 'windows-nt)
                  (format "%s:#%d" (file-truename buffer-file-name) (1- (position-bytes point)))
                (format "%s:#%d" (shell-quote-argument (file-truename buffer-file-name)) (1- (position-bytes point)))))
        (out (godoc--get-buffer "<all>")))
  (with-current-buffer (get-buffer-create "*go-getdef-input*")
    (setq-local inhibit-eol-conversion t)
    (setq buffer-read-only nil)
    (erase-buffer)
    (go--insert-modified-files buff)
    (call-process-region (point-min) (point-max) "gogetdef" nil out nil
                         "-modified"
                         "-all"
                         (format "-pos=%s" posn)))
  (with-current-buffer out
    (goto-char (point-min))
    (process-methods)
    (godoc-mode)
    (display-buffer (current-buffer) t))))

(defun process-methods()
  (let (beg end (methods ()) method)
    (when (re-search-forward "\\[\\(:method:\\)\\(\\[[^|]*|[^|]*\\]\\)+\\]" nil t)
      (setq beg (match-beginning 0) end (match-end 0))
      (goto-char (match-end 1))
      (while (re-search-forward "\\[\\([^|]*\\)|\\([^]|]*\\)\\]" nil t)
        (add-to-list 'methods (list (match-string 1) (match-string 2)) t))
      (delete-region beg end)
      (while methods
        (setq method (pop methods))
        (insert-text-button (nth 0 method) 'type 'gogetdef-button 'args (nth 1 method))
        (insert "\n"))
      (goto-char (point-min)))))

(define-button-type 'gogetdef-button
  'follow-link t
  'action (lambda (botton)
            (let ((pos (button-get botton 'args)))
              (godef--find-file-line-column pos t))))

(provide 'gogetdef)
