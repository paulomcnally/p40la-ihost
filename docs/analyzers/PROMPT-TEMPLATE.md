# Prompt Template para LLM Externo

Copia y pega esto cuando le pidas al LLM externo que genere un analyzer.

---

## Prompt

```
Eres un desarrollador Go. Tu tarea es crear un "analyzer" para extraer datos de facturas.

CONTEXTO:
- El proyecto usa Go 1.24 con un sistema de analyzers basado en registry + init()
- Los analyzers estan en internal/analyzers/all/
- Cada analyzer implementa la interfaz DocumentAnalyzer
- Extraen: invoice_number, amount, year, month, due_date

ARCHIVOS DE REFERENCIA (leelos):
- internal/analyzers/analyzer.go (interfaz)
- internal/analyzers/all/claro.go (ejemplo completo)

INSTRUCCIONES:
1. Lee el PDF/imagen que te envio
2. Identifica los campos: numero de factura, monto total, periodo facturado, fecha de vencimiento
3. Genera el archivo Go del analyzer en internal/analyzers/all/[tu_id].go
4. Genera los tests en internal/analyzers/all/[tu_id]_test.go
5. El ID debe ser snake_case unico (ej: "tigo_hogar", "cable_color")
6. Usa regexp para extraer cada campo del texto plano del PDF
7. Valida que todos los campos obligatorios esten presentes antes de devolver

CAMPOS OBLIGATORIOS:
- InvoiceNumber (string) - numero unico de factura
- Amount (float64) - monto total a pagar
- Year (int) - anio de la factura
- Month (int) - mes 1-12

CAMPOS OPCIONALES:
- DueDate (*time.Time) - fecha limite de pago
- RawData (map[string]interface{}) - datos extra

ENTREGA:
1. Archivo [tu_id].go completo y funcional
2. Archivo [tu_id]_test.go con tests usando texto real extraido del PDF
3. El texto plano que extrajiste del PDF (para que yo pueda crear mas tests)
```

---

## Ejemplo de uso

1. Sube la imagen/PDF de la factura al LLM externo
2. Pega el prompt de arriba
3. El LLM te da el codigo
4. Copia los archivos a `internal/analyzers/all/`
5. Ejecuta `go test ./internal/analyzers/all/` para validar
6. Si pasa, el analyzer esta listo
