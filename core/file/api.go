package file

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

type entryCreate struct {
	Path string `json:"path" validate:"required"`
	Type string `json:"type" validate:"required,oneof=file directory"`
}

type contentWrite struct {
	Content *string `json:"content" validate:"required,max=2097152"`
}

type entryPatch struct {
	Destination *string `json:"destination" validate:"omitempty,min=1"`
	Permissions *string `json:"permissions" validate:"omitempty,min=3,max=4"`
}

type fileEntry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	Permissions string    `json:"permissions"`
	ModifiedAt  time.Time `json:"modified_at"`
}

type fileResource struct {
	fileEntry
	Content *string      `json:"content,omitempty"`
	Entries *[]fileEntry `json:"entries,omitempty"`
}

type uploadRequest struct {
	Files []string `json:"files" validate:"required"`
}

var permissionsPattern = regexp.MustCompile(`^[0-7]{3,4}$`)

func (handler *Handler) RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/files/*", handler.getEntry, httpapi.Operation{ID: "getFile", Summary: "Read a directory listing or UTF-8 file content (up to 2 MiB); logical path is rooted at the file manager root", Response: fileResource{}})
	router.Handle(http.MethodPost, "/files", handler.createEntry, httpapi.Operation{ID: "createFile", Summary: "Create a new file or directory without overwriting an existing resource", Status: http.StatusCreated, Request: entryCreate{}, Response: fileEntry{}})
	router.Handle(http.MethodPut, "/files/*", handler.putContent, httpapi.Operation{ID: "writeFile", Summary: "Replace existing file content with UTF-8 text", Request: contentWrite{}, Response: fileEntry{}})
	router.Handle(http.MethodPatch, "/files/*", handler.patchEntry, httpapi.Operation{ID: "patchFile", Summary: "Set exactly one of destination (rename) or permissions (octal string)", Request: entryPatch{}, Response: fileEntry{}})
	router.Handle(http.MethodDelete, "/files/*", handler.deleteEntry, httpapi.Operation{ID: "deleteFile", Summary: "Recursively delete a file or directory; deleting the root is forbidden", Response: map[string]string{}})
	router.Handle(http.MethodGet, "/downloads/*", handler.download, httpapi.Operation{ID: "downloadFile", Summary: "Download raw file bytes instead of JSON", ResponseMedia: "application/octet-stream", Response: ""})
	router.Handle(http.MethodPost, "/uploads/*", handler.upload, httpapi.Operation{ID: "uploadFiles", Summary: "Upload multipart files into a directory, replacing same-name files; total body limit is 32 MiB", Request: uploadRequest{}, RequestMedia: "multipart/form-data", Response: map[string][]string{}})
}

func logicalPath(ctx *echo.Context, resource string) string {
	return strings.TrimPrefix(ctx.Request().URL.Path, httpapi.Prefix+resource)
}

func metadata(logicalPath string, info os.FileInfo) fileEntry {
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	return fileEntry{
		Name: info.Name(), Path: normalizeLogicalPath(logicalPath), Type: kind, Size: info.Size(),
		Permissions: fmtPermissions(info), ModifiedAt: info.ModTime(),
	}
}

func fmtPermissions(info os.FileInfo) string {
	return fmt.Sprintf("%04o", info.Mode().Perm())
}

func (handler *Handler) getEntry(ctx *echo.Context) error {
	requestedPath := logicalPath(ctx, "/files")
	info, err := handler.service.Stat(requestedPath)
	if err != nil {
		return err
	}
	resource := fileResource{fileEntry: metadata(requestedPath, info)}
	if info.IsDir() {
		entries, err := handler.service.ReadDir(requestedPath)
		if err != nil {
			return err
		}
		sort.SliceStable(entries, func(first, second int) bool {
			if entries[first].IsDir() != entries[second].IsDir() {
				return entries[first].IsDir()
			}
			return entries[first].Name() < entries[second].Name()
		})
		listing := make([]fileEntry, 0, len(entries))
		for _, entry := range entries {
			listing = append(listing, metadata(path.Join(requestedPath, entry.Name()), entry))
		}
		resource.Entries = &listing
	} else {
		content, err := handler.service.ReadEditable(requestedPath)
		if err != nil {
			return err
		}
		resource.Content = &content
	}
	return httpapi.OK(ctx, resource)
}

func rejectRoot(requestedPath string) error {
	if normalizeLogicalPath(requestedPath) == "/" {
		return httpapi.NewError(http.StatusUnprocessableEntity, "root_protected", "Modifying the filesystem root is forbidden")
	}
	return nil
}

func (handler *Handler) respondMetadata(ctx *echo.Context, requestedPath string, status int) error {
	info, err := handler.service.Stat(requestedPath)
	if err != nil {
		return err
	}
	return httpapi.Respond(ctx, status, metadata(requestedPath, info))
}

func (handler *Handler) createEntry(ctx *echo.Context) error {
	var request entryCreate
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if err := rejectRoot(request.Path); err != nil {
		return err
	}
	if _, err := handler.service.Stat(request.Path); err == nil {
		return httpapi.NewError(http.StatusConflict, "already_exists", "File or directory already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var err error
	if request.Type == "directory" {
		err = handler.service.CreateFolder(request.Path)
	} else {
		err = handler.service.CreateFile(request.Path)
	}
	if err != nil {
		return err
	}
	return handler.respondMetadata(ctx, request.Path, http.StatusCreated)
}

func (handler *Handler) putContent(ctx *echo.Context) error {
	var request contentWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if len(*request.Content) > 2<<20 {
		return httpapi.NewError(http.StatusRequestEntityTooLarge, "body_too_large", "File content exceeds 2 MiB")
	}
	requestedPath := logicalPath(ctx, "/files")
	if err := rejectRoot(requestedPath); err != nil {
		return err
	}
	if _, err := handler.service.Stat(requestedPath); err != nil {
		return err
	}
	if err := handler.service.WriteFile(requestedPath, []byte(*request.Content)); err != nil {
		return err
	}
	return handler.respondMetadata(ctx, requestedPath, http.StatusOK)
}

func (handler *Handler) patchEntry(ctx *echo.Context) error {
	var request entryPatch
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if (request.Destination == nil) == (request.Permissions == nil) {
		return httpapi.NewError(http.StatusUnprocessableEntity, "validation_failed", "Set exactly one of destination or permissions")
	}
	requestedPath := logicalPath(ctx, "/files")
	if err := rejectRoot(requestedPath); err != nil {
		return err
	}
	if request.Destination != nil {
		if err := rejectRoot(*request.Destination); err != nil {
			return err
		}
		if _, err := handler.service.Stat(*request.Destination); err == nil {
			return httpapi.NewError(http.StatusConflict, "already_exists", "Destination already exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := handler.service.Rename(requestedPath, *request.Destination); err != nil {
			return err
		}
		requestedPath = *request.Destination
	} else {
		permissions, err := strconv.ParseUint(*request.Permissions, 8, 32)
		if err != nil || !permissionsPattern.MatchString(*request.Permissions) || permissions > 0777 {
			return httpapi.NewError(http.StatusUnprocessableEntity, "invalid_permissions", "permissions must be an octal string between 0000 and 0777")
		}
		if err := handler.service.Chmod(requestedPath, *request.Permissions); err != nil {
			return err
		}
	}
	return handler.respondMetadata(ctx, requestedPath, http.StatusOK)
}

func (handler *Handler) deleteEntry(ctx *echo.Context) error {
	requestedPath := logicalPath(ctx, "/files")
	if err := rejectRoot(requestedPath); err != nil {
		return err
	}
	if _, err := handler.service.Stat(requestedPath); err != nil {
		return err
	}
	if err := handler.service.RemoveAll(requestedPath); err != nil {
		return err
	}
	return httpapi.OK(ctx, map[string]string{"path": normalizeLogicalPath(requestedPath)})
}

func (handler *Handler) download(ctx *echo.Context) error {
	requestedPath := logicalPath(ctx, "/downloads")
	info, err := handler.service.Stat(requestedPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return httpapi.NewError(http.StatusUnprocessableEntity, "not_a_file", "Only files can be downloaded")
	}
	file, err := handler.service.Open(requestedPath)
	if err != nil {
		return err
	}
	defer file.Close()
	ctx.Response().Header().Set(echo.HeaderContentDisposition, mime.FormatMediaType("attachment", map[string]string{"filename": path.Base(requestedPath)}))
	return ctx.Stream(http.StatusOK, "application/octet-stream", file)
}

func (handler *Handler) upload(ctx *echo.Context) error {
	mediaType, _, err := mime.ParseMediaType(ctx.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != "multipart/form-data" {
		return httpapi.NewError(http.StatusUnsupportedMediaType, "unsupported_media_type", "Use multipart/form-data with a files field")
	}
	ctx.Request().Body = http.MaxBytesReader(ctx.Response(), ctx.Request().Body, 32<<20)
	if err := ctx.Request().ParseMultipartForm(4 << 20); err != nil {
		var oversized *http.MaxBytesError
		if errors.As(err, &oversized) {
			return httpapi.NewError(http.StatusRequestEntityTooLarge, "body_too_large", "Upload body exceeds 32 MiB")
		}
		return httpapi.NewError(http.StatusBadRequest, "invalid_upload", err.Error())
	}
	defer ctx.Request().MultipartForm.RemoveAll()
	headers := ctx.Request().MultipartForm.File["files"]
	if len(headers) == 0 {
		return httpapi.NewError(http.StatusUnprocessableEntity, "validation_failed", "At least one file is required")
	}
	directory := logicalPath(ctx, "/uploads")
	uploaded := make([]string, 0, len(headers))
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return err
		}
		err = handler.service.SaveUploadedFile(directory, header.Filename, file)
		file.Close()
		if err != nil {
			return err
		}
		uploaded = append(uploaded, path.Join(directory, header.Filename))
	}
	return httpapi.OK(ctx, map[string][]string{"uploaded": uploaded})
}
