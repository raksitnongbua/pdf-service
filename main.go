package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4/middleware"

	"github.com/labstack/echo/v4"
	"github.com/unidoc/unipdf/v3/common/license"
	"github.com/unidoc/unipdf/v3/creator"
	"github.com/unidoc/unipdf/v3/model"
	"github.com/joho/godotenv"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func uploadFileHandler(c echo.Context) error {
	_, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	file, _, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("./uploads", "temp.pdf")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	// TODO: the password must be encrypted and add the logic decrypt here.
	password := c.Request().FormValue("password")
	fmt.Println("password: ", password)

	err = decryptPDF(tempFile.Name(), password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}
	return c.File(tempFile.Name())
}

func decryptPDF(pathFile, password string) error {
	file, err := os.Open(pathFile)
	if err != nil {
		return fmt.Errorf("error opening encrypted PDF file: %v", err)
	}
	defer file.Close()

	pdfReader, err := model.NewPdfReader(file)
	if err != nil {
		return fmt.Errorf("error creating PDF reader: %v", err)
	}

	authenticated, err := pdfReader.Decrypt([]byte(password))
	if err != nil {
		return fmt.Errorf("error decrypting PDF: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("authentication failed: incorrect password")
	}

	pdfWriter := creator.New()

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return fmt.Errorf("error getting number of pages: %v", err)
	}
	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return fmt.Errorf("error getting PDF page: %v", err)
		}
		pdfWriter.AddPage(page)
	}

	err = pdfWriter.WriteToFile(pathFile)
	if err != nil {
		return fmt.Errorf("error writing decrypted PDF to file: %v", err)
	}

	return nil
}

// TODO: split middleware and pdf unlock service.
func main() {
	err := os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		fmt.Println("Error creating uploads directory:", err)
		return
	}


	godotenv.Load(".env")
	metredKey := os.Getenv("UNIDOC_METERED_LICENSE_KEY")
	license.SetMeteredKey(metredKey)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	e.GET("/api", func(c echo.Context) error {
		return c.String(http.StatusOK, "api is ready!")
	})
	
	e.POST("/api/unlock-pdf", uploadFileHandler)

	e.Logger.Fatal(e.Start(":8080"))
}
