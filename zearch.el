(require 'web)
(require 'json)

;; uses *zearch* for  the results, and *zearch-fetch* for the files
;; i use it like this so it searches either for the selection or for the current word:
;; (key-chord-define-global "jj" 'zearch-search-current)

(defun zearch-json-get (url)
  (with-current-buffer (url-retrieve-synchronously url)
    (goto-char (point-min))
    (re-search-forward "^$")
    (json-read)))

(defun zearch-http-get (url)
  (with-current-buffer (url-retrieve-synchronously url)
    (goto-char (point-min))
    (search-forward "\n\n")
    (delete-region (point-min) (point))
    (delete-region (point) (point-min))
    (buffer-string)))

(defun zearch-fetch (id)
  (kill-buffer (get-buffer-create "*zearch-fetch*"))
  (with-current-buffer (get-buffer-create "*zearch-fetch*")
    (insert (zearch-http-get (format "http://localhost:8080/fetch?%s" id)))
    (goto-char (point-min)))
  (switch-to-buffer "*zearch-fetch*"))

(defun zearch-search (query)
  (kill-buffer (get-buffer-create "*zearch*"))
  (with-current-buffer (get-buffer-create "*zearch*")
    (insert (format "results for: %s" query))
    (newline)
    (let ((xurl (format "http://localhost:8080/search?%s" query)))
      (let ((hits (assoc-default 'Hits (zearch-json-get xurl))))
        (dotimes (i (length hits))
          (let ((hit (elt hits i)))
            (let ((path (assoc-default 'Path hit))
                  (id (assoc-default 'Id hit))
                  (score (assoc-default 'Score hit)))
              (insert (format "%s | s:%d | %06d" path score id))
              (newline))))))
    (zearch-mode)
    (goto-char (point-min)))
  (switch-to-buffer "*zearch*"))

(defun zearch-search-current ()
  (interactive)
  (if mark-active
      (progn
        (let ((from (region-beginning)))
          (let ((to (region-end)))
            (zearch-search (buffer-substring from to)))))
    (progn
      (zearch-search (thing-at-point 'word t)))))

(defun zearch-mode ()
  (let ((map (make-sparse-keymap))
        (fetch (lambda ()
                 (interactive)
                 (let ((line (thing-at-point 'line t)))
                   (zearch-fetch (substring line -7 -1)))))) ;; last 6 digits are the id
    (define-key map (kbd "RET") fetch) 
    (define-key map "\C-j" fetch)
    (define-key map "\C-m" fetch)
    (use-local-map map)
    (setq major-mode 'zearch-mode)
    (setq mode-name "zearch")))

(provide 'zearch)
