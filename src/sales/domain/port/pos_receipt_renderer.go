package port

import (
	"sales/src/sales/application/response"
)

// PosReceiptTenant es la información mínima del comercio que se imprime en el
// encabezado del comprobante. Se mantiene desacoplada de TenantInfo para que el
// renderer del dominio no dependa de detalles del adaptador de tenants.
type PosReceiptTenant struct {
	Name string
}

// PosReceiptRenderer es un driven port: dado el detalle de una venta POS y los
// datos del comercio, produce los bytes de un comprobante renderizable
// (E18 Tramo C: PDF A4 INTERNO, no fiscal).
//
// El use case orquesta (obtiene el detalle vía GetPosSaleUseCase) y delega el
// render a este port; la implementación concreta vive en infrastructure/adapter.
type PosReceiptRenderer interface {
	// RenderPDF retorna el comprobante como un PDF A4 listo para servir
	// (Content-Type application/pdf). detail nunca debe ser nil.
	RenderPDF(detail *response.POSSaleDetailResponse, tenant PosReceiptTenant) ([]byte, error)
}
