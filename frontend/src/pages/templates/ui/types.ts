export interface DynamicFieldMapping {
  columnIndex?: number;
  sourceColumn?: string;
  required?: boolean;
  defaultValue?: string;
}

export interface DynamicTemplateRules {
  format: "csv" | "zip";
  hasHeader: boolean;
  mapping: Record<string, DynamicFieldMapping>;
  extraData: {
    strategy: "catch_all" | "explicit";
    explicitMapping?: Record<string, DynamicFieldMapping>;
  };
}

export interface ExportColumnMapping {
  headerName: string;
  valueType:
    | "order_no"
    | "recipient"
    | "phone"
    | "address"
    | "sku"
    | "quantity"
    | "static";
  prefix?: string;
  defaultValue?: string;
}

export interface DynamicExportRules {
  format: string;
  hasHeader: boolean;
  columns: ExportColumnMapping[];
}
