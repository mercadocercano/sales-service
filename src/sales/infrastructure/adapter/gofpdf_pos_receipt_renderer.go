package adapter

import (
	"bytes"
	"fmt"

	"sales/src/sales/application/response"
	"sales/src/sales/domain/port"

	"github.com/jung-kurt/gofpdf"
)

// GofpdfPosReceiptRenderer implementa port.PosReceiptRenderer usando gofpdf.
// E18 Tramo C: comprobante POS INTERNO en A4 (no fiscal/AFIP).
//
// Se eligió github.com/jung-kurt/gofpdf por ser Go puro (sin CGO ni
// dependencias pesadas de fuentes/imágenes), determinista y suficiente para un
// comprobante de una página con encabezado + tabla + totales.
type GofpdfPosReceiptRenderer struct{}

// NewGofpdfPosReceiptRenderer crea el adapter de render de comprobantes.
func NewGofpdfPosReceiptRenderer() *GofpdfPosReceiptRenderer {
	return &GofpdfPosReceiptRenderer{}
}

const (
	pageMarginLeft = 15.0
	pageWidthA4    = 210.0
	contentWidth   = pageWidthA4 - 2*pageMarginLeft // 180mm
)

// RenderPDF construye el PDF A4 del comprobante.
func (r *GofpdfPosReceiptRenderer) RenderPDF(detail *response.POSSaleDetailResponse, tenant port.PosReceiptTenant) ([]byte, error) {
	if detail == nil {
		return nil, fmt.Errorf("pos receipt: detail is nil")
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pageMarginLeft, 15, pageMarginLeft)
	pdf.AddPage()

	r.writeHeader(pdf, detail, tenant)
	r.writeItemsTable(pdf, detail)
	r.writeTotals(pdf, detail)
	r.writeFooter(pdf, detail)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pos receipt: output pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func (r *GofpdfPosReceiptRenderer) writeHeader(pdf *gofpdf.Fpdf, detail *response.POSSaleDetailResponse, tenant port.PosReceiptTenant) {
	commerce := tenant.Name
	if commerce == "" {
		commerce = "Comercio"
	}

	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(contentWidth, 10, commerce, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(contentWidth, 6, "Comprobante de venta (documento no fiscal)", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	number := "—"
	if detail.SaleNumber != nil {
		number = fmt.Sprintf("#%d", *detail.SaleNumber)
	}

	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(contentWidth/2, 6, fmt.Sprintf("Comprobante N°: %s", number), "", 0, "L", false, 0, "")
	pdf.CellFormat(contentWidth/2, 6, fmt.Sprintf("Fecha: %s", detail.CreatedAt.Format("02/01/2006 15:04")), "", 1, "R", false, 0, "")

	// El nombre del cliente ya viene resuelto en el detalle (use case):
	// "Consumidor Final" para ventas sin cliente, el nombre real, o el fallback.
	customer := detail.CustomerName
	if customer == "" {
		customer = "Consumidor Final"
	}
	pdf.CellFormat(contentWidth, 6, fmt.Sprintf("Cliente: %s", customer), "", 1, "L", false, 0, "")
	pdf.Ln(4)
}

func (r *GofpdfPosReceiptRenderer) writeItemsTable(pdf *gofpdf.Fpdf, detail *response.POSSaleDetailResponse) {
	// Anchos de columna (mm) — suman contentWidth (180).
	const (
		wSKU      = 30.0
		wName     = 78.0
		wQty      = 18.0
		wUnit     = 27.0
		wSubtotal = 27.0
	)

	// Encabezado de tabla
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(wSKU, 8, "SKU", "1", 0, "L", true, 0, "")
	pdf.CellFormat(wName, 8, "Producto", "1", 0, "L", true, 0, "")
	pdf.CellFormat(wQty, 8, "Cant.", "1", 0, "R", true, 0, "")
	pdf.CellFormat(wUnit, 8, "P. Unit.", "1", 0, "R", true, 0, "")
	pdf.CellFormat(wSubtotal, 8, "Subtotal", "1", 1, "R", true, 0, "")

	// Filas
	pdf.SetFont("Helvetica", "", 10)
	for _, it := range detail.Items {
		name := it.ProductName
		if len(name) > 48 {
			name = name[:45] + "..."
		}
		pdf.CellFormat(wSKU, 7, it.SKU, "1", 0, "L", false, 0, "")
		pdf.CellFormat(wName, 7, name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(wQty, 7, fmt.Sprintf("%d", it.Quantity), "1", 0, "R", false, 0, "")
		pdf.CellFormat(wUnit, 7, money(detail.Currency, it.UnitPrice.String()), "1", 0, "R", false, 0, "")
		pdf.CellFormat(wSubtotal, 7, money(detail.Currency, it.Subtotal.String()), "1", 1, "R", false, 0, "")
	}
	pdf.Ln(4)
}

func (r *GofpdfPosReceiptRenderer) writeTotals(pdf *gofpdf.Fpdf, detail *response.POSSaleDetailResponse) {
	const (
		wLabel = 130.0
		wValue = 50.0
	)
	pdf.SetFont("Helvetica", "", 11)

	row := func(label, value string, bold bool) {
		style := ""
		if bold {
			style = "B"
		}
		pdf.SetFont("Helvetica", style, 11)
		pdf.CellFormat(wLabel, 7, label, "", 0, "R", false, 0, "")
		pdf.CellFormat(wValue, 7, value, "", 1, "R", false, 0, "")
	}

	row("Subtotal:", money(detail.Currency, detail.SubtotalAmount.String()), false)
	row("Descuento:", "-"+money(detail.Currency, detail.DiscountAmount.String()), false)
	row("TOTAL:", money(detail.Currency, detail.FinalAmount.String()), true)
	row("Pagado:", money(detail.Currency, detail.AmountPaid.String()), false)
	row("Vuelto:", money(detail.Currency, detail.Change.String()), false)
	pdf.Ln(4)
}

func (r *GofpdfPosReceiptRenderer) writeFooter(pdf *gofpdf.Fpdf, detail *response.POSSaleDetailResponse) {
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(contentWidth, 7, fmt.Sprintf("Medio de pago: %s", detail.PaymentMethodName), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(contentWidth, 6, "Gracias por su compra.", "", 1, "L", false, 0, "")
}

// money formatea un importe con el código de moneda. Se asume el valor ya
// expresado como decimal en string (p. ej. "180.00").
func money(currency, amount string) string {
	if currency == "" {
		currency = "ARS"
	}
	return fmt.Sprintf("%s %s", currency, amount)
}
