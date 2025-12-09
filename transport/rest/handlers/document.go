package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
	"github.com/tasklineby/certify-backend/service"
)

type DocumentHandler struct {
	documentService service.DocumentService
}

func NewDocumentHandler(documentService service.DocumentService) *DocumentHandler {
	return &DocumentHandler{documentService: documentService}
}

// getCompanyIDFromContext extracts company_id from the gin context (set by auth middleware)
func getCompanyIDFromContext(c *gin.Context) (int, error) {
	companyIDStr, exists := c.Get("company_id")
	if !exists {
		return 0, errs.UnauthorizedError("company ID not found in context", nil)
	}

	companyID, err := strconv.Atoi(companyIDStr.(string))
	if err != nil {
		return 0, errs.InternalError("invalid company ID in token", err)
	}

	return companyID, nil
}

// getUserIDFromContext extracts user_id from the gin context (set by auth middleware)
func getUserIDFromContext(c *gin.Context) (int, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, errs.UnauthorizedError("user ID not found in context", nil)
	}

	return userID.(int), nil
}

// CreateDocument godoc
// @Summary      Create a document
// @Description  Create a new document for the authenticated user's company with PDF file attachment
// @Tags         documents
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        type            formData  string  true  "Document type"
// @Param        name            formData  string  true  "Document name"
// @Param        summary         formData  string  true  "Document summary"
// @Param        expiration_date formData  string  true  "Expiration date (RFC3339 format)"
// @Param        file            formData  file    true  "PDF file"
// @Success      201       {object}  entity.CreateDocumentResponse  "Document created successfully"
// @Failure      400       {object}  errs.Error                     "Invalid request"
// @Failure      401       {object}  errs.Error                     "Unauthorized"
// @Failure      500       {object}  errs.Error                     "Internal server error"
// @Router       /documents [post]
func (h *DocumentHandler) CreateDocument(c *gin.Context) {
	companyID, err := getCompanyIDFromContext(c)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	// Parse form data
	docType := c.PostForm("type")
	name := c.PostForm("name")
	summary := c.PostForm("summary")
	expirationDateStr := c.PostForm("expiration_date")

	if docType == "" || name == "" || summary == "" || expirationDateStr == "" {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("type, name, summary and expiration_date are required", nil))
		return
	}

	expirationDate, err := time.Parse(time.RFC3339, expirationDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid expiration_date format, use RFC3339", err))
		return
	}

	req := entity.CreateDocumentRequest{
		Type:           docType,
		Name:           name,
		Summary:        summary,
		ExpirationDate: expirationDate,
	}

	// Handle mandatory file upload
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("file is required", err))
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errs.InternalError("failed to read file", err))
		return
	}

	fileName := header.Filename

	hash, err := h.documentService.CreateDocument(c.Request.Context(), req, companyID, fileName, fileData)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusCreated, entity.CreateDocumentResponse{Hash: hash})
}

// DownloadFile godoc
// @Summary      Download document file
// @Description  Download the PDF file attached to a document. Only employees from the same company can access.
// @Tags         documents
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        id        path      int  true  "Document ID"
// @Success      200       {file}    binary           "PDF file"
// @Failure      400       {object}  errs.Error       "Invalid document ID"
// @Failure      401       {object}  errs.Error       "Unauthorized"
// @Failure      404       {object}  errs.Error       "Document or file not found"
// @Failure      500       {object}  errs.Error       "Internal server error"
// @Router       /documents/{id}/file [get]
func (h *DocumentHandler) DownloadFile(c *gin.Context) {
	companyID, err := getCompanyIDFromContext(c)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid document ID", err))
		return
	}

	doc, err := h.documentService.GetDocumentByID(c.Request.Context(), id, companyID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", doc.FileName))
	c.Header("Content-Type", "application/pdf")
	c.Data(http.StatusOK, "application/pdf", doc.FileData)
}

// GetCompanyDocuments godoc
// @Summary      Get all company documents
// @Description  Get all documents for the authenticated user's company
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Success      200       {array}   entity.Document  "List of company documents"
// @Failure      401       {object}  errs.Error       "Unauthorized"
// @Failure      500       {object}  errs.Error       "Internal server error"
// @Router       /documents [get]
func (h *DocumentHandler) GetCompanyDocuments(c *gin.Context) {
	companyID, err := getCompanyIDFromContext(c)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	docs, err := h.documentService.GetDocumentsByCompanyID(c.Request.Context(), companyID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, docs)
}

// GetDocument godoc
// @Summary      Get document by ID
// @Description  Get a document by its ID. Only employees from the same company can access.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      int  true  "Document ID"
// @Success      200       {object}  entity.Document  "Document details"
// @Failure      400       {object}  errs.Error       "Invalid document ID"
// @Failure      401       {object}  errs.Error       "Unauthorized"
// @Failure      404       {object}  errs.Error       "Document not found"
// @Failure      500       {object}  errs.Error       "Internal server error"
// @Router       /documents/{id} [get]
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	companyID, err := getCompanyIDFromContext(c)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid document ID", err))
		return
	}

	doc, err := h.documentService.GetDocumentByID(c.Request.Context(), id, companyID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, doc)
}

// VerifyDocument godoc
// @Summary      Verify a document by hash
// @Description  Verify a document using its hash from query parameter and get full details with expiration status. Only employees from the same company can verify. Each verification is recorded in history.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        hash      query     string  true  "Document hash"
// @Success      200       {object}  entity.VerifyDocumentResponse  "Document verification result"
// @Failure      400       {object}  errs.Error                     "Invalid request or hash"
// @Failure      401       {object}  errs.Error                     "Unauthorized"
// @Failure      500       {object}  errs.Error                     "Internal server error"
// @Router       /documents/verify [get]
func (h *DocumentHandler) VerifyDocument(c *gin.Context) {
	companyID, err := getCompanyIDFromContext(c)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	hash := c.Query("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("hash query parameter is required", nil))
		return
	}

	doc, status, message, err := h.documentService.VerifyDocument(c.Request.Context(), hash, companyID, userID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, entity.VerifyDocumentResponse{
		Document: doc,
		Status:   status,
		Message:  message,
	})
}

// GetHistory godoc
// @Summary      Get verification history
// @Description  Get the authenticated user's document verification history
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Success      200       {array}   entity.VerificationHistory  "Verification history"
// @Failure      401       {object}  errs.Error                  "Unauthorized"
// @Failure      500       {object}  errs.Error                  "Internal server error"
// @Router       /history [get]
func (h *DocumentHandler) GetHistory(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	history, err := h.documentService.GetHistory(c.Request.Context(), userID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, history)
}
