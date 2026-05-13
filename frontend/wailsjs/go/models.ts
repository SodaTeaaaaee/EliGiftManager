export namespace dto {
	
	export class CreateDemandLineInput {
	    lineType: string;
	    obligationTriggerKind: string;
	    entitlementAuthority: string;
	    routingDisposition: string;
	    externalTitle: string;
	    requestedQuantity: number;
	
	    static createFrom(source: any = {}) {
	        return new CreateDemandLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lineType = source["lineType"];
	        this.obligationTriggerKind = source["obligationTriggerKind"];
	        this.entitlementAuthority = source["entitlementAuthority"];
	        this.routingDisposition = source["routingDisposition"];
	        this.externalTitle = source["externalTitle"];
	        this.requestedQuantity = source["requestedQuantity"];
	    }
	}
	export class CreateDemandInput {
	    kind: string;
	    captureMode: string;
	    sourceChannel: string;
	    sourceDocumentNo: string;
	    lines: CreateDemandLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateDemandInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.captureMode = source["captureMode"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceDocumentNo = source["sourceDocumentNo"];
	        this.lines = this.convertValues(source["lines"], CreateDemandLineInput);
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
	
	export class CreateWaveInput {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateWaveInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class DemandDocumentDTO {
	    id: number;
	    kind: string;
	    captureMode: string;
	    sourceChannel: string;
	    sourceSurface: string;
	    integrationProfileId?: number;
	    sourceDocumentNo: string;
	    sourceCustomerRef: string;
	    customerProfileId?: number;
	    sourceCreatedAt: string;
	    sourcePaidAt: string;
	    currency: string;
	    authoritySnapshotAt: string;
	    rawPayload: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new DemandDocumentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.captureMode = source["captureMode"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceSurface = source["sourceSurface"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.sourceDocumentNo = source["sourceDocumentNo"];
	        this.sourceCustomerRef = source["sourceCustomerRef"];
	        this.customerProfileId = source["customerProfileId"];
	        this.sourceCreatedAt = source["sourceCreatedAt"];
	        this.sourcePaidAt = source["sourcePaidAt"];
	        this.currency = source["currency"];
	        this.authoritySnapshotAt = source["authoritySnapshotAt"];
	        this.rawPayload = source["rawPayload"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class DemandLineDTO {
	    id: number;
	    demandDocumentId: number;
	    sourceLineNo?: number;
	    lineType: string;
	    obligationTriggerKind: string;
	    entitlementAuthority: string;
	    recipientInputState: string;
	    routingDisposition: string;
	    routingReasonCode: string;
	    eligibilityContextRef: string;
	    productMasterId?: number;
	    externalTitle: string;
	    requestedQuantity: number;
	    entitlementCode: string;
	    giftLevelSnapshot: string;
	    recipientInputPayload: string;
	    rawPayload: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new DemandLineDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.demandDocumentId = source["demandDocumentId"];
	        this.sourceLineNo = source["sourceLineNo"];
	        this.lineType = source["lineType"];
	        this.obligationTriggerKind = source["obligationTriggerKind"];
	        this.entitlementAuthority = source["entitlementAuthority"];
	        this.recipientInputState = source["recipientInputState"];
	        this.routingDisposition = source["routingDisposition"];
	        this.routingReasonCode = source["routingReasonCode"];
	        this.eligibilityContextRef = source["eligibilityContextRef"];
	        this.productMasterId = source["productMasterId"];
	        this.externalTitle = source["externalTitle"];
	        this.requestedQuantity = source["requestedQuantity"];
	        this.entitlementCode = source["entitlementCode"];
	        this.giftLevelSnapshot = source["giftLevelSnapshot"];
	        this.recipientInputPayload = source["recipientInputPayload"];
	        this.rawPayload = source["rawPayload"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class FulfillmentLineDTO {
	    id: number;
	    waveId: number;
	    customerProfileId?: number;
	    waveParticipantSnapshotId?: number;
	    productId?: number;
	    demandDocumentId?: number;
	    demandLineId?: number;
	    customerAddressId?: number;
	    quantity: number;
	    allocationState: string;
	    addressState: string;
	    supplierState: string;
	    channelSyncState: string;
	    lineReason: string;
	    generatedBy: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new FulfillmentLineDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.customerProfileId = source["customerProfileId"];
	        this.waveParticipantSnapshotId = source["waveParticipantSnapshotId"];
	        this.productId = source["productId"];
	        this.demandDocumentId = source["demandDocumentId"];
	        this.demandLineId = source["demandLineId"];
	        this.customerAddressId = source["customerAddressId"];
	        this.quantity = source["quantity"];
	        this.allocationState = source["allocationState"];
	        this.addressState = source["addressState"];
	        this.supplierState = source["supplierState"];
	        this.channelSyncState = source["channelSyncState"];
	        this.lineReason = source["lineReason"];
	        this.generatedBy = source["generatedBy"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SupplierOrderDTO {
	    id: number;
	    waveId: number;
	    supplierPlatform: string;
	    templateId: string;
	    batchNo: string;
	    externalOrderNo: string;
	    submissionMode: string;
	    submittedAt: string;
	    status: string;
	    requestPayload: string;
	    responsePayload: string;
	    basisHistoryNodeId: string;
	    basisProjectionHash: string;
	    basisPayloadSnapshot: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SupplierOrderDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.supplierPlatform = source["supplierPlatform"];
	        this.templateId = source["templateId"];
	        this.batchNo = source["batchNo"];
	        this.externalOrderNo = source["externalOrderNo"];
	        this.submissionMode = source["submissionMode"];
	        this.submittedAt = source["submittedAt"];
	        this.status = source["status"];
	        this.requestPayload = source["requestPayload"];
	        this.responsePayload = source["responsePayload"];
	        this.basisHistoryNodeId = source["basisHistoryNodeId"];
	        this.basisProjectionHash = source["basisProjectionHash"];
	        this.basisPayloadSnapshot = source["basisPayloadSnapshot"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class WaveDTO {
	    id: number;
	    waveNo: string;
	    name: string;
	    waveType: string;
	    lifecycleStage: string;
	    progressSnapshot: string;
	    notes: string;
	    levelTags: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new WaveDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveNo = source["waveNo"];
	        this.name = source["name"];
	        this.waveType = source["waveType"];
	        this.lifecycleStage = source["lifecycleStage"];
	        this.progressSnapshot = source["progressSnapshot"];
	        this.notes = source["notes"];
	        this.levelTags = source["levelTags"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class WaveOverviewDTO {
	    wave: WaveDTO;
	    demandCount: number;
	    fulfillmentCount: number;
	    supplierOrderCount: number;
	
	    static createFrom(source: any = {}) {
	        return new WaveOverviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wave = this.convertValues(source["wave"], WaveDTO);
	        this.demandCount = source["demandCount"];
	        this.fulfillmentCount = source["fulfillmentCount"];
	        this.supplierOrderCount = source["supplierOrderCount"];
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

