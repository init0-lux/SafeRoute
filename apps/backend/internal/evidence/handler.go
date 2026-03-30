package evidence

import (
	"errors"

	"saferoute-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service   *Service
	authGuard fiber.Handler
}

type metadataResponse struct {
	Evidence *Metadata `json:"evidence"`
}

func NewHandler(service *Service, authGuard fiber.Handler) *Handler {
	return &Handler{
		service:   service,
		authGuard: authGuard,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/evidence/upload", h.authGuard, h.upload)
	router.Get("/evidence/:id", h.authGuard, h.getByID)
	router.Get("/evidence/:id/content", h.authGuard, h.download)
}

func (h *Handler) upload(c *fiber.Ctx) error {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		return writeEvidenceError(c, auth.ErrUnauthorized)
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return writeEvidenceError(c, ErrFileRequired)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open upload"})
	}

	metadata, err := h.service.Upload(c.UserContext(), UploadInput{
		UserID:    currentUser.ID,
		ReportID:  c.FormValue("report_id"),
		SessionID: c.FormValue("session_id"),
		File:      file,
		Filename:  fileHeader.Filename,
		Size:      fileHeader.Size,
	}, c.BaseURL())
	if err != nil {
		return writeEvidenceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(metadataResponse{Evidence: metadata})
}

func (h *Handler) getByID(c *fiber.Ctx) error {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		return writeEvidenceError(c, auth.ErrUnauthorized)
	}

	metadata, err := h.service.GetByID(c.UserContext(), c.Params("id"), currentUser.ID, c.BaseURL())
	if err != nil {
		return writeEvidenceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(metadataResponse{Evidence: metadata})
}

func (h *Handler) download(c *fiber.Ctx) error {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		return writeEvidenceError(c, auth.ErrUnauthorized)
	}

	download, err := h.service.Download(c.UserContext(), c.Params("id"), currentUser.ID, c.BaseURL())
	if err != nil {
		return writeEvidenceError(c, err)
	}

	c.Set("Content-Type", download.Metadata.MediaType)
	if download.Metadata.OriginalFilename != "" {
		c.Attachment(download.Metadata.OriginalFilename)
	}

	return c.SendStream(download.Reader)
}

func writeEvidenceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, auth.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrFileRequired),
		errors.Is(err, ErrFileTooLarge),
		errors.Is(err, ErrUnsupportedMediaType),
		errors.Is(err, ErrInvalidEvidenceID),
		errors.Is(err, ErrInvalidReportID),
		errors.Is(err, ErrInvalidSessionID),
		errors.Is(err, ErrAttachmentTargetRequired),
		errors.Is(err, ErrAttachmentTargetConflict):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrEvidenceNotFound), errors.Is(err, ErrParentNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrForbiddenEvidenceAccess):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}
