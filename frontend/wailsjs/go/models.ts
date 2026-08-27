export namespace alignment {
	
	export class ParseIssue {
	    LineNo: number;
	    Key: string;
	    Message: string;
	
	    static createFrom(source: any = {}) {
	        return new ParseIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.LineNo = source["LineNo"];
	        this.Key = source["Key"];
	        this.Message = source["Message"];
	    }
	}
	export class TemplatePreview {
	    Rows: any[];
	    Issues: ParseIssue[];
	
	    static createFrom(source: any = {}) {
	        return new TemplatePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Rows = source["Rows"];
	        this.Issues = this.convertValues(source["Issues"], ParseIssue);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace app {
	
	export class ExceptionView {
	    ID: number;
	    InstanceID: number;
	    CustomerName: string;
	    ProductItemID: number;
	    ProductName: string;
	    Quantity: number;
	    Note: string;
	
	    static createFrom(source: any = {}) {
	        return new ExceptionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.InstanceID = source["InstanceID"];
	        this.CustomerName = source["CustomerName"];
	        this.ProductItemID = source["ProductItemID"];
	        this.ProductName = source["ProductName"];
	        this.Quantity = source["Quantity"];
	        this.Note = source["Note"];
	    }
	}
	export class ExportFileResult {
	    Order: domain.SupplierOrder;
	    Path: string;
	    Rows: any[];
	
	    static createFrom(source: any = {}) {
	        return new ExportFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Order = this.convertValues(source["Order"], domain.SupplierOrder);
	        this.Path = source["Path"];
	        this.Rows = source["Rows"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GenerateFactoryOrderResult {
	    Order: domain.SupplierOrder;
	    Lines: domain.SupplierOrderLine[];
	
	    static createFrom(source: any = {}) {
	        return new GenerateFactoryOrderResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Order = this.convertValues(source["Order"], domain.SupplierOrder);
	        this.Lines = this.convertValues(source["Lines"], domain.SupplierOrderLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HomeBuckets {
	    Unassigned: number;
	    DuplicateAsk: number;
	    AlignmentConflict: number;
	    IdentityUnattached: number;
	    PendingRevisions: number;
	    BlockedResults: number;
	    WritebackFailed: number;
	    ResidualClose: number;
	    RevisionFrozenConflicts: number;
	    RecentWaves: domain.Wave[];
	
	    static createFrom(source: any = {}) {
	        return new HomeBuckets(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Unassigned = source["Unassigned"];
	        this.DuplicateAsk = source["DuplicateAsk"];
	        this.AlignmentConflict = source["AlignmentConflict"];
	        this.IdentityUnattached = source["IdentityUnattached"];
	        this.PendingRevisions = source["PendingRevisions"];
	        this.BlockedResults = source["BlockedResults"];
	        this.WritebackFailed = source["WritebackFailed"];
	        this.ResidualClose = source["ResidualClose"];
	        this.RevisionFrozenConflicts = source["RevisionFrozenConflicts"];
	        this.RecentWaves = this.convertValues(source["RecentWaves"], domain.Wave);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportFileResult {
	    Document: domain.InputDocument;
	    FactsCreated: number;
	    LinesCreated: number;
	    Duplicates: domain.DuplicateObservation[];
	    Issues: alignment.ParseIssue[];
	
	    static createFrom(source: any = {}) {
	        return new ImportFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Document = this.convertValues(source["Document"], domain.InputDocument);
	        this.FactsCreated = source["FactsCreated"];
	        this.LinesCreated = source["LinesCreated"];
	        this.Duplicates = this.convertValues(source["Duplicates"], domain.DuplicateObservation);
	        this.Issues = this.convertValues(source["Issues"], alignment.ParseIssue);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SkippedShipment {
	    LineNo: number;
	    TrackingID: string;
	    Reason: string;
	
	    static createFrom(source: any = {}) {
	        return new SkippedShipment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.LineNo = source["LineNo"];
	        this.TrackingID = source["TrackingID"];
	        this.Reason = source["Reason"];
	    }
	}
	export class ImportShipmentFileResult {
	    Imported: number;
	    Skipped: SkippedShipment[];
	    Shipments: domain.Shipment[];
	    Issues: alignment.ParseIssue[];
	
	    static createFrom(source: any = {}) {
	        return new ImportShipmentFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Imported = source["Imported"];
	        this.Skipped = this.convertValues(source["Skipped"], SkippedShipment);
	        this.Shipments = this.convertValues(source["Shipments"], domain.Shipment);
	        this.Issues = this.convertValues(source["Issues"], alignment.ParseIssue);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InboxRow {
	    Line: domain.InputFactLine;
	    Fact: domain.InputFact;
	    Document?: domain.InputDocument;
	    Assigned: boolean;
	    Unaligned: boolean;
	    Unattached: boolean;
	    RevisionPending: boolean;
	    AliasID?: number;
	
	    static createFrom(source: any = {}) {
	        return new InboxRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Line = this.convertValues(source["Line"], domain.InputFactLine);
	        this.Fact = this.convertValues(source["Fact"], domain.InputFact);
	        this.Document = this.convertValues(source["Document"], domain.InputDocument);
	        this.Assigned = source["Assigned"];
	        this.Unaligned = source["Unaligned"];
	        this.Unattached = source["Unattached"];
	        this.RevisionPending = source["RevisionPending"];
	        this.AliasID = source["AliasID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IngestDocumentResult {
	    Document: domain.InputDocument;
	    Duplicates: domain.DuplicateObservation[];
	
	    static createFrom(source: any = {}) {
	        return new IngestDocumentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Document = this.convertValues(source["Document"], domain.InputDocument);
	        this.Duplicates = this.convertValues(source["Duplicates"], domain.DuplicateObservation);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IngestLine {
	    SourceLineNo: number;
	    ExternalSKU: string;
	    ExternalTitle: string;
	    ExternalSpec: string;
	    Quantity: number;
	
	    static createFrom(source: any = {}) {
	        return new IngestLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SourceLineNo = source["SourceLineNo"];
	        this.ExternalSKU = source["ExternalSKU"];
	        this.ExternalTitle = source["ExternalTitle"];
	        this.ExternalSpec = source["ExternalSpec"];
	        this.Quantity = source["Quantity"];
	    }
	}
	export class IngestFactInput {
	    Kind: string;
	    StableExternalID: string;
	    IdentityType: string;
	    IdentityValue: string;
	    MembershipLevel: string;
	    SourceDocumentNo: string;
	    // Go type: time
	    SourceCreatedAt?: any;
	    CustomerProfileID?: number;
	    ExtraData: string;
	    Lines: IngestLine[];
	
	    static createFrom(source: any = {}) {
	        return new IngestFactInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Kind = source["Kind"];
	        this.StableExternalID = source["StableExternalID"];
	        this.IdentityType = source["IdentityType"];
	        this.IdentityValue = source["IdentityValue"];
	        this.MembershipLevel = source["MembershipLevel"];
	        this.SourceDocumentNo = source["SourceDocumentNo"];
	        this.SourceCreatedAt = this.convertValues(source["SourceCreatedAt"], null);
	        this.CustomerProfileID = source["CustomerProfileID"];
	        this.ExtraData = source["ExtraData"];
	        this.Lines = this.convertValues(source["Lines"], IngestLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class InstanceView {
	    ID: number;
	    CustomerName: string;
	    PlatformIdentity: string;
	    MembershipLevel: string;
	
	    static createFrom(source: any = {}) {
	        return new InstanceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.CustomerName = source["CustomerName"];
	        this.PlatformIdentity = source["PlatformIdentity"];
	        this.MembershipLevel = source["MembershipLevel"];
	    }
	}
	export class ProductTotal {
	    ProductID: number;
	    Name: string;
	    Quantity: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProductID = source["ProductID"];
	        this.Name = source["Name"];
	        this.Quantity = source["Quantity"];
	    }
	}
	export class ResultView {
	    Result: domain.FulfillmentResult;
	    WorkState: string;
	    Blocks: string[];
	    InFactory: boolean;
	    Shipped: boolean;
	    WritebackFailed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResultView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Result = this.convertValues(source["Result"], domain.FulfillmentResult);
	        this.WorkState = source["WorkState"];
	        this.Blocks = source["Blocks"];
	        this.InFactory = source["InFactory"];
	        this.Shipped = source["Shipped"];
	        this.WritebackFailed = source["WritebackFailed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace domain {
	
	export class AddressSnapshot {
	    source_address_id?: number;
	    recipient_name: string;
	    phone: string;
	    country: string;
	    province: string;
	    city: string;
	    district: string;
	    address_line1: string;
	    address_line2: string;
	    postal_code: string;
	
	    static createFrom(source: any = {}) {
	        return new AddressSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_address_id = source["source_address_id"];
	        this.recipient_name = source["recipient_name"];
	        this.phone = source["phone"];
	        this.country = source["country"];
	        this.province = source["province"];
	        this.city = source["city"];
	        this.district = source["district"];
	        this.address_line1 = source["address_line1"];
	        this.address_line2 = source["address_line2"];
	        this.postal_code = source["postal_code"];
	    }
	}
	export class AppSettings {
	    ID: number;
	    Locale: string;
	    Theme: string;
	    Density: string;
	    DuplicateRecordMinutes: number;
	    DuplicateAskDays: number;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Locale = source["Locale"];
	        this.Theme = source["Theme"];
	        this.Density = source["Density"];
	        this.DuplicateRecordMinutes = source["DuplicateRecordMinutes"];
	        this.DuplicateAskDays = source["DuplicateAskDays"];
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CarrierMapping {
	    ID: number;
	    PlatformID: number;
	    ExternalCode: string;
	    InternalCode: string;
	    InternalName: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new CarrierMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.PlatformID = source["PlatformID"];
	        this.ExternalCode = source["ExternalCode"];
	        this.InternalCode = source["InternalCode"];
	        this.InternalName = source["InternalName"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChannelWritebackItem {
	    ID: number;
	    InputFactID: number;
	    ShipmentID: number;
	    TrackingNo: string;
	    CarrierCode: string;
	    Quantity: number;
	    Status: string;
	    TemplateID: number;
	    TemplateVersion: number;
	    RetryCount: number;
	    ErrorMessage: string;
	    Payload: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ChannelWritebackItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.InputFactID = source["InputFactID"];
	        this.ShipmentID = source["ShipmentID"];
	        this.TrackingNo = source["TrackingNo"];
	        this.CarrierCode = source["CarrierCode"];
	        this.Quantity = source["Quantity"];
	        this.Status = source["Status"];
	        this.TemplateID = source["TemplateID"];
	        this.TemplateVersion = source["TemplateVersion"];
	        this.RetryCount = source["RetryCount"];
	        this.ErrorMessage = source["ErrorMessage"];
	        this.Payload = source["Payload"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerProfile {
	    ID: number;
	    DisplayName: string;
	    Notes: string;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new CustomerProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.DisplayName = source["DisplayName"];
	        this.Notes = source["Notes"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DuplicateObservation {
	    ID: number;
	    DocumentID: number;
	    ExistingFactID: number;
	    Verdict: string;
	    Reason: string;
	    Decided: boolean;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateObservation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.DocumentID = source["DocumentID"];
	        this.ExistingFactID = source["ExistingFactID"];
	        this.Verdict = source["Verdict"];
	        this.Reason = source["Reason"];
	        this.Decided = source["Decided"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EntitlementException {
	    ID: number;
	    WaveID: number;
	    ProductID: number;
	    InstanceID: number;
	    Quantity: number;
	    Note: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new EntitlementException(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.WaveID = source["WaveID"];
	        this.ProductID = source["ProductID"];
	        this.InstanceID = source["InstanceID"];
	        this.Quantity = source["Quantity"];
	        this.Note = source["Note"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EntitlementSelector {
	    type: string;
	    platform_id?: number;
	    level?: string;
	    instance_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new EntitlementSelector(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.platform_id = source["platform_id"];
	        this.level = source["level"];
	        this.instance_id = source["instance_id"];
	    }
	}
	export class EntitlementRule {
	    ID: number;
	    WaveID: number;
	    ProductID: number;
	    Selector: EntitlementSelector;
	    Quantity: number;
	    Active: boolean;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new EntitlementRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.WaveID = source["WaveID"];
	        this.ProductID = source["ProductID"];
	        this.Selector = this.convertValues(source["Selector"], EntitlementSelector);
	        this.Quantity = source["Quantity"];
	        this.Active = source["Active"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FulfillmentResult {
	    ID: number;
	    WaveID: number;
	    SourceKind: string;
	    EntitlementInstanceID?: number;
	    InputFactLineID?: number;
	    InputFactID?: number;
	    CustomerProfileID?: number;
	    ProductItemID?: number;
	    Quantity: number;
	    Address: AddressSnapshot;
	    Frozen: boolean;
	    AddressPinned: boolean;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new FulfillmentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.WaveID = source["WaveID"];
	        this.SourceKind = source["SourceKind"];
	        this.EntitlementInstanceID = source["EntitlementInstanceID"];
	        this.InputFactLineID = source["InputFactLineID"];
	        this.InputFactID = source["InputFactID"];
	        this.CustomerProfileID = source["CustomerProfileID"];
	        this.ProductItemID = source["ProductItemID"];
	        this.Quantity = source["Quantity"];
	        this.Address = this.convertValues(source["Address"], AddressSnapshot);
	        this.Frozen = source["Frozen"];
	        this.AddressPinned = source["AddressPinned"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InputDocument {
	    ID: number;
	    PlatformID: number;
	    DocumentType: string;
	    Direction: string;
	    OriginalName: string;
	    RawPayload: string;
	    TemplateID?: number;
	    TemplateVersion: number;
	    // Go type: time
	    ImportedAt: any;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new InputDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.PlatformID = source["PlatformID"];
	        this.DocumentType = source["DocumentType"];
	        this.Direction = source["Direction"];
	        this.OriginalName = source["OriginalName"];
	        this.RawPayload = source["RawPayload"];
	        this.TemplateID = source["TemplateID"];
	        this.TemplateVersion = source["TemplateVersion"];
	        this.ImportedAt = this.convertValues(source["ImportedAt"], null);
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InputFact {
	    ID: number;
	    DocumentID?: number;
	    PlatformID: number;
	    Kind: string;
	    StableExternalID: string;
	    CustomerProfileID?: number;
	    PlatformIdentityID?: number;
	    MembershipLevel: string;
	    SourceDocumentNo: string;
	    // Go type: time
	    SourceCreatedAt?: any;
	    RevisesID?: number;
	    // Go type: time
	    RevisionAppliedAt?: any;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new InputFact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.DocumentID = source["DocumentID"];
	        this.PlatformID = source["PlatformID"];
	        this.Kind = source["Kind"];
	        this.StableExternalID = source["StableExternalID"];
	        this.CustomerProfileID = source["CustomerProfileID"];
	        this.PlatformIdentityID = source["PlatformIdentityID"];
	        this.MembershipLevel = source["MembershipLevel"];
	        this.SourceDocumentNo = source["SourceDocumentNo"];
	        this.SourceCreatedAt = this.convertValues(source["SourceCreatedAt"], null);
	        this.RevisesID = source["RevisesID"];
	        this.RevisionAppliedAt = this.convertValues(source["RevisionAppliedAt"], null);
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InputFactLine {
	    ID: number;
	    FactID: number;
	    SourceLineNo: number;
	    ExternalSKU: string;
	    ExternalTitle: string;
	    ExternalSpec: string;
	    ProductItemID?: number;
	    Quantity: number;
	    WaveID?: number;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new InputFactLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.FactID = source["FactID"];
	        this.SourceLineNo = source["SourceLineNo"];
	        this.ExternalSKU = source["ExternalSKU"];
	        this.ExternalTitle = source["ExternalTitle"];
	        this.ExternalSpec = source["ExternalSpec"];
	        this.ProductItemID = source["ProductItemID"];
	        this.Quantity = source["Quantity"];
	        this.WaveID = source["WaveID"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Platform {
	    ID: number;
	    Key: string;
	    Name: string;
	    Kind: string;
	    Notes: string;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Platform(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Key = source["Key"];
	        this.Name = source["Name"];
	        this.Kind = source["Kind"];
	        this.Notes = source["Notes"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProductAlias {
	    ID: number;
	    ProductItemID: number;
	    PlatformID: number;
	    ExternalProductID: string;
	    Title: string;
	    Spec: string;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ProductAlias(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.ProductItemID = source["ProductItemID"];
	        this.PlatformID = source["PlatformID"];
	        this.ExternalProductID = source["ExternalProductID"];
	        this.Title = source["Title"];
	        this.Spec = source["Spec"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProductBundleComponent {
	    ID: number;
	    AliasID: number;
	    ProductItemID: number;
	    Quantity: number;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ProductBundleComponent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.AliasID = source["AliasID"];
	        this.ProductItemID = source["ProductItemID"];
	        this.Quantity = source["Quantity"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProductItem {
	    ID: number;
	    Name: string;
	    FactoryPlatformID: number;
	    FactorySKU: string;
	    Notes: string;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ProductItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.FactoryPlatformID = source["FactoryPlatformID"];
	        this.FactorySKU = source["FactorySKU"];
	        this.Notes = source["Notes"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class QuantitySplitComponent {
	    ID: number;
	    RuleID: number;
	    ProductItemID: number;
	    Quantity: number;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new QuantitySplitComponent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.RuleID = source["RuleID"];
	        this.ProductItemID = source["ProductItemID"];
	        this.Quantity = source["Quantity"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class QuantitySplitRule {
	    ID: number;
	    WaveID: number;
	    PlatformID: number;
	    ExternalKey: string;
	    Components: QuantitySplitComponent[];
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new QuantitySplitRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.WaveID = source["WaveID"];
	        this.PlatformID = source["PlatformID"];
	        this.ExternalKey = source["ExternalKey"];
	        this.Components = this.convertValues(source["Components"], QuantitySplitComponent);
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RecipientAddress {
	    ID: number;
	    CustomerProfileID: number;
	    Label: string;
	    RecipientName: string;
	    Phone: string;
	    Country: string;
	    Province: string;
	    City: string;
	    District: string;
	    AddressLine1: string;
	    AddressLine2: string;
	    PostalCode: string;
	    IsDefault: boolean;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new RecipientAddress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.CustomerProfileID = source["CustomerProfileID"];
	        this.Label = source["Label"];
	        this.RecipientName = source["RecipientName"];
	        this.Phone = source["Phone"];
	        this.Country = source["Country"];
	        this.Province = source["Province"];
	        this.City = source["City"];
	        this.District = source["District"];
	        this.AddressLine1 = source["AddressLine1"];
	        this.AddressLine2 = source["AddressLine2"];
	        this.PostalCode = source["PostalCode"];
	        this.IsDefault = source["IsDefault"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Shipment {
	    ID: number;
	    TrackingID: string;
	    CarrierCode: string;
	    CarrierName: string;
	    TrackingNo: string;
	    // Go type: time
	    ShippedAt?: any;
	    Quantity: number;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Shipment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.TrackingID = source["TrackingID"];
	        this.CarrierCode = source["CarrierCode"];
	        this.CarrierName = source["CarrierName"];
	        this.TrackingNo = source["TrackingNo"];
	        this.ShippedAt = this.convertValues(source["ShippedAt"], null);
	        this.Quantity = source["Quantity"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierOrder {
	    ID: number;
	    WaveID: number;
	    FactoryPlatformID: number;
	    Status: string;
	    TemplateID: number;
	    TemplateVersion: number;
	    // Go type: time
	    ExportedAt?: any;
	    // Go type: time
	    VoidedAt?: any;
	    ExportPayload: string;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new SupplierOrder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.WaveID = source["WaveID"];
	        this.FactoryPlatformID = source["FactoryPlatformID"];
	        this.Status = source["Status"];
	        this.TemplateID = source["TemplateID"];
	        this.TemplateVersion = source["TemplateVersion"];
	        this.ExportedAt = this.convertValues(source["ExportedAt"], null);
	        this.VoidedAt = this.convertValues(source["VoidedAt"], null);
	        this.ExportPayload = source["ExportPayload"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierOrderLine {
	    ID: number;
	    SupplierOrderID: number;
	    ProductItemID: number;
	    FactorySKU: string;
	    Quantity: number;
	    TrackingID: string;
	    TrackingRetired: boolean;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new SupplierOrderLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.SupplierOrderID = source["SupplierOrderID"];
	        this.ProductItemID = source["ProductItemID"];
	        this.FactorySKU = source["FactorySKU"];
	        this.Quantity = source["Quantity"];
	        this.TrackingID = source["TrackingID"];
	        this.TrackingRetired = source["TrackingRetired"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TemplateConfig {
	    ID: number;
	    PlatformID: number;
	    DocumentType: string;
	    Direction: string;
	    Name: string;
	    Version: number;
	    Builtin: boolean;
	    MappingJSON: string;
	    LayoutJSON: string;
	    Notes: string;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new TemplateConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.PlatformID = source["PlatformID"];
	        this.DocumentType = source["DocumentType"];
	        this.Direction = source["Direction"];
	        this.Name = source["Name"];
	        this.Version = source["Version"];
	        this.Builtin = source["Builtin"];
	        this.MappingJSON = source["MappingJSON"];
	        this.LayoutJSON = source["LayoutJSON"];
	        this.Notes = source["Notes"];
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Wave {
	    ID: number;
	    WaveNo: string;
	    Name: string;
	    Notes: string;
	    CloseResult: string;
	    CloseNote: string;
	    // Go type: time
	    ClosedAt?: any;
	    // Go type: time
	    ReopenedAt?: any;
	    ExtraData: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Wave(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.WaveNo = source["WaveNo"];
	        this.Name = source["Name"];
	        this.Notes = source["Notes"];
	        this.CloseResult = source["CloseResult"];
	        this.CloseNote = source["CloseNote"];
	        this.ClosedAt = this.convertValues(source["ClosedAt"], null);
	        this.ReopenedAt = this.convertValues(source["ReopenedAt"], null);
	        this.ExtraData = source["ExtraData"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

