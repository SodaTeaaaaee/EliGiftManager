export namespace domain {
	
	export class SelectorPayload {
	    type: string;
	    platform?: string;
	    level?: string;
	    participant_ids?: number[];
	
	    static createFrom(source: any = {}) {
	        return new SelectorPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.platform = source["platform"];
	        this.level = source["level"];
	        this.participant_ids = source["participant_ids"];
	    }
	}

}

export namespace dto {
	
	export class AllocationPolicyRuleDTO {
	    id: number;
	    wave_id: number;
	    product_id: number;
	    selector_payload: domain.SelectorPayload;
	    product_target_ref: string;
	    contribution_quantity: number;
	    rule_kind: string;
	    priority: number;
	    active: boolean;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new AllocationPolicyRuleDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.wave_id = source["wave_id"];
	        this.product_id = source["product_id"];
	        this.selector_payload = this.convertValues(source["selector_payload"], domain.SelectorPayload);
	        this.product_target_ref = source["product_target_ref"];
	        this.contribution_quantity = source["contribution_quantity"];
	        this.rule_kind = source["rule_kind"];
	        this.priority = source["priority"];
	        this.active = source["active"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
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
	export class BasisDriftSignalDTO {
	    basisKind: string;
	    basisDriftStatus: string;
	    reviewRequirement: string;
	    driftReasonCodes: string[];
	
	    static createFrom(source: any = {}) {
	        return new BasisDriftSignalDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.basisKind = source["basisKind"];
	        this.basisDriftStatus = source["basisDriftStatus"];
	        this.reviewRequirement = source["reviewRequirement"];
	        this.driftReasonCodes = source["driftReasonCodes"];
	    }
	}
	export class BindTemplateToProfileInput {
	    integrationProfileId: number;
	    documentType: string;
	    templateId: number;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BindTemplateToProfileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.integrationProfileId = source["integrationProfileId"];
	        this.documentType = source["documentType"];
	        this.templateId = source["templateId"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class ChannelSyncItemDTO {
	    id: number;
	    channelSyncJobId: number;
	    fulfillmentLineId: number;
	    shipmentId: number;
	    externalDocumentNo: string;
	    externalLineNo: string;
	    trackingNo: string;
	    carrierCode: string;
	    status: string;
	    errorMessage: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ChannelSyncItemDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.channelSyncJobId = source["channelSyncJobId"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.shipmentId = source["shipmentId"];
	        this.externalDocumentNo = source["externalDocumentNo"];
	        this.externalLineNo = source["externalLineNo"];
	        this.trackingNo = source["trackingNo"];
	        this.carrierCode = source["carrierCode"];
	        this.status = source["status"];
	        this.errorMessage = source["errorMessage"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ChannelSyncJobDTO {
	    id: number;
	    waveId: number;
	    integrationProfileId: number;
	    direction: string;
	    status: string;
	    basisHistoryNodeId: string;
	    basisProjectionHash: string;
	    basisPayloadSnapshot: string;
	    requestPayload: string;
	    responsePayload: string;
	    errorMessage: string;
	    startedAt: string;
	    finishedAt: string;
	    createdAt: string;
	    updatedAt: string;
	    items: ChannelSyncItemDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ChannelSyncJobDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.direction = source["direction"];
	        this.status = source["status"];
	        this.basisHistoryNodeId = source["basisHistoryNodeId"];
	        this.basisProjectionHash = source["basisProjectionHash"];
	        this.basisPayloadSnapshot = source["basisPayloadSnapshot"];
	        this.requestPayload = source["requestPayload"];
	        this.responsePayload = source["responsePayload"];
	        this.errorMessage = source["errorMessage"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.items = this.convertValues(source["items"], ChannelSyncItemDTO);
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
	export class ClosureDecisionRecordDTO {
	    id: number;
	    waveId: number;
	    integrationProfileId: number;
	    fulfillmentLineId: number;
	    decisionKind: string;
	    reasonCode: string;
	    note: string;
	    evidenceRef: string;
	    operatorId: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ClosureDecisionRecordDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.decisionKind = source["decisionKind"];
	        this.reasonCode = source["reasonCode"];
	        this.note = source["note"];
	        this.evidenceRef = source["evidenceRef"];
	        this.operatorId = source["operatorId"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class CreateAllocationPolicyRuleInput {
	    wave_id: number;
	    product_id: number;
	    selector_payload: domain.SelectorPayload;
	    product_target_ref: string;
	    contribution_quantity: number;
	    rule_kind: string;
	    priority: number;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateAllocationPolicyRuleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wave_id = source["wave_id"];
	        this.product_id = source["product_id"];
	        this.selector_payload = this.convertValues(source["selector_payload"], domain.SelectorPayload);
	        this.product_target_ref = source["product_target_ref"];
	        this.contribution_quantity = source["contribution_quantity"];
	        this.rule_kind = source["rule_kind"];
	        this.priority = source["priority"];
	        this.active = source["active"];
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
	export class CreateChannelSyncItemInput {
	    fulfillmentLineId: number;
	    shipmentId: number;
	    externalDocumentNo: string;
	    externalLineNo: string;
	    trackingNo: string;
	    carrierCode: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateChannelSyncItemInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.shipmentId = source["shipmentId"];
	        this.externalDocumentNo = source["externalDocumentNo"];
	        this.externalLineNo = source["externalLineNo"];
	        this.trackingNo = source["trackingNo"];
	        this.carrierCode = source["carrierCode"];
	    }
	}
	export class CreateChannelSyncJobInput {
	    waveId: number;
	    integrationProfileId: number;
	    direction: string;
	    items: CreateChannelSyncItemInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateChannelSyncJobInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.direction = source["direction"];
	        this.items = this.convertValues(source["items"], CreateChannelSyncItemInput);
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
	export class CreateDemandLineInput {
	    lineType: string;
	    obligationTriggerKind: string;
	    entitlementAuthority: string;
	    recipientInputState: string;
	    routingDisposition: string;
	    routingReasonCode: string;
	    eligibilityContextRef: string;
	    entitlementCode: string;
	    giftLevelSnapshot: string;
	    productMasterId?: number;
	    recipientInputPayload: string;
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
	        this.recipientInputState = source["recipientInputState"];
	        this.routingDisposition = source["routingDisposition"];
	        this.routingReasonCode = source["routingReasonCode"];
	        this.eligibilityContextRef = source["eligibilityContextRef"];
	        this.entitlementCode = source["entitlementCode"];
	        this.giftLevelSnapshot = source["giftLevelSnapshot"];
	        this.productMasterId = source["productMasterId"];
	        this.recipientInputPayload = source["recipientInputPayload"];
	        this.externalTitle = source["externalTitle"];
	        this.requestedQuantity = source["requestedQuantity"];
	    }
	}
	export class CreateDemandInput {
	    kind: string;
	    captureMode: string;
	    sourceChannel: string;
	    sourceSurface: string;
	    sourceDocumentNo: string;
	    sourceCustomerRef: string;
	    customerProfileId?: number;
	    integrationProfileId?: number;
	    lines: CreateDemandLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateDemandInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.captureMode = source["captureMode"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceSurface = source["sourceSurface"];
	        this.sourceDocumentNo = source["sourceDocumentNo"];
	        this.sourceCustomerRef = source["sourceCustomerRef"];
	        this.customerProfileId = source["customerProfileId"];
	        this.integrationProfileId = source["integrationProfileId"];
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
	
	export class CreateDocumentTemplateInput {
	    templateKey: string;
	    documentType: string;
	    format: string;
	    mappingRules: string;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateDocumentTemplateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.templateKey = source["templateKey"];
	        this.documentType = source["documentType"];
	        this.format = source["format"];
	        this.mappingRules = source["mappingRules"];
	        this.extraData = source["extraData"];
	    }
	}
	export class CreateProductMasterInput {
	    supplierPlatform: string;
	    factorySku: string;
	    supplierProductRef: string;
	    name: string;
	    productKind: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateProductMasterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplierPlatform = source["supplierPlatform"];
	        this.factorySku = source["factorySku"];
	        this.supplierProductRef = source["supplierProductRef"];
	        this.name = source["name"];
	        this.productKind = source["productKind"];
	    }
	}
	export class CreateProfileInput {
	    profileKey: string;
	    sourceChannel: string;
	    sourceSurface: string;
	    demandKind: string;
	    initialAllocationStrategy: string;
	    identityStrategy: string;
	    entitlementAuthorityMode: string;
	    recipientInputMode: string;
	    referenceStrategy: string;
	    trackingSyncMode: string;
	    closurePolicy: string;
	    supportsPartialShipment: boolean;
	    supportsApiImport: boolean;
	    supportsApiExport: boolean;
	    requiresCarrierMapping: boolean;
	    requiresExternalOrderNo: boolean;
	    allowsManualClosure: boolean;
	    connectorKey: string;
	    supportedLocales: string;
	    defaultLocale: string;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateProfileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileKey = source["profileKey"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceSurface = source["sourceSurface"];
	        this.demandKind = source["demandKind"];
	        this.initialAllocationStrategy = source["initialAllocationStrategy"];
	        this.identityStrategy = source["identityStrategy"];
	        this.entitlementAuthorityMode = source["entitlementAuthorityMode"];
	        this.recipientInputMode = source["recipientInputMode"];
	        this.referenceStrategy = source["referenceStrategy"];
	        this.trackingSyncMode = source["trackingSyncMode"];
	        this.closurePolicy = source["closurePolicy"];
	        this.supportsPartialShipment = source["supportsPartialShipment"];
	        this.supportsApiImport = source["supportsApiImport"];
	        this.supportsApiExport = source["supportsApiExport"];
	        this.requiresCarrierMapping = source["requiresCarrierMapping"];
	        this.requiresExternalOrderNo = source["requiresExternalOrderNo"];
	        this.allowsManualClosure = source["allowsManualClosure"];
	        this.connectorKey = source["connectorKey"];
	        this.supportedLocales = source["supportedLocales"];
	        this.defaultLocale = source["defaultLocale"];
	        this.extraData = source["extraData"];
	    }
	}
	export class CreateShipmentLineInput {
	    supplierOrderLineId: number;
	    fulfillmentLineId: number;
	    quantity: number;
	
	    static createFrom(source: any = {}) {
	        return new CreateShipmentLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplierOrderLineId = source["supplierOrderLineId"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.quantity = source["quantity"];
	    }
	}
	export class CreateShipmentInput {
	    supplierOrderId: number;
	    supplierPlatform: string;
	    shipmentNo: string;
	    externalShipmentNo: string;
	    carrierCode: string;
	    carrierName: string;
	    trackingNo: string;
	    status: string;
	    shippedAt: string;
	    basisPayloadSnapshot: string;
	    lines: CreateShipmentLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateShipmentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplierOrderId = source["supplierOrderId"];
	        this.supplierPlatform = source["supplierPlatform"];
	        this.shipmentNo = source["shipmentNo"];
	        this.externalShipmentNo = source["externalShipmentNo"];
	        this.carrierCode = source["carrierCode"];
	        this.carrierName = source["carrierName"];
	        this.trackingNo = source["trackingNo"];
	        this.status = source["status"];
	        this.shippedAt = source["shippedAt"];
	        this.basisPayloadSnapshot = source["basisPayloadSnapshot"];
	        this.lines = this.convertValues(source["lines"], CreateShipmentLineInput);
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
	export class DemandInboxFilterInput {
	    assignment: string;
	    demandKind: string;
	
	    static createFrom(source: any = {}) {
	        return new DemandInboxFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assignment = source["assignment"];
	        this.demandKind = source["demandKind"];
	    }
	}
	export class DemandInboxRowDTO {
	    demandDocumentId: number;
	    kind: string;
	    captureMode: string;
	    sourceChannel: string;
	    sourceSurface: string;
	    sourceDocumentNo: string;
	    customerProfileId?: number;
	    integrationProfileId?: number;
	    integrationProfileLabel: string;
	    assigned: boolean;
	    assignedWaveId?: number;
	    assignedWaveLabel: string;
	    totalLineCount: number;
	    acceptedCount: number;
	    readyAcceptedCount: number;
	    waitingInputCount: number;
	    deferredCount: number;
	    excludedCount: number;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new DemandInboxRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demandDocumentId = source["demandDocumentId"];
	        this.kind = source["kind"];
	        this.captureMode = source["captureMode"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceSurface = source["sourceSurface"];
	        this.sourceDocumentNo = source["sourceDocumentNo"];
	        this.customerProfileId = source["customerProfileId"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.integrationProfileLabel = source["integrationProfileLabel"];
	        this.assigned = source["assigned"];
	        this.assignedWaveId = source["assignedWaveId"];
	        this.assignedWaveLabel = source["assignedWaveLabel"];
	        this.totalLineCount = source["totalLineCount"];
	        this.acceptedCount = source["acceptedCount"];
	        this.readyAcceptedCount = source["readyAcceptedCount"];
	        this.waitingInputCount = source["waitingInputCount"];
	        this.deferredCount = source["deferredCount"];
	        this.excludedCount = source["excludedCount"];
	        this.createdAt = source["createdAt"];
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
	export class DemandMappingBlockedLine {
	    demandLineId: number;
	    demandLineTitle: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new DemandMappingBlockedLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demandLineId = source["demandLineId"];
	        this.demandLineTitle = source["demandLineTitle"];
	        this.reason = source["reason"];
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
	export class DemandMappingResult {
	    createdLines: FulfillmentLineDTO[];
	    blockedLines: DemandMappingBlockedLine[];
	
	    static createFrom(source: any = {}) {
	        return new DemandMappingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.createdLines = this.convertValues(source["createdLines"], FulfillmentLineDTO);
	        this.blockedLines = this.convertValues(source["blockedLines"], DemandMappingBlockedLine);
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
	export class DocumentTemplateDTO {
	    id: number;
	    templateKey: string;
	    documentType: string;
	    format: string;
	    mappingRules: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentTemplateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.templateKey = source["templateKey"];
	        this.documentType = source["documentType"];
	        this.format = source["format"];
	        this.mappingRules = source["mappingRules"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ExecuteSyncResult {
	    jobId: number;
	    jobStatus: string;
	    requestPayload: string;
	    responsePayload: string;
	    errorMessage: string;
	    startedAt: string;
	    finishedAt: string;
	    items: ChannelSyncItemDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ExecuteSyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.jobStatus = source["jobStatus"];
	        this.requestPayload = source["requestPayload"];
	        this.responsePayload = source["responsePayload"];
	        this.errorMessage = source["errorMessage"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.items = this.convertValues(source["items"], ChannelSyncItemDTO);
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
	export class FulfillmentAdjustmentDTO {
	    id: number;
	    waveId: number;
	    targetKind: string;
	    fulfillmentLineId?: number;
	    waveParticipantSnapshotId?: number;
	    adjustmentKind: string;
	    quantityDelta: number;
	    reasonCode: string;
	    operatorId: string;
	    note: string;
	    evidenceRef: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new FulfillmentAdjustmentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.targetKind = source["targetKind"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.waveParticipantSnapshotId = source["waveParticipantSnapshotId"];
	        this.adjustmentKind = source["adjustmentKind"];
	        this.quantityDelta = source["quantityDelta"];
	        this.reasonCode = source["reasonCode"];
	        this.operatorId = source["operatorId"];
	        this.note = source["note"];
	        this.evidenceRef = source["evidenceRef"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	
	export class HistoryNodeDTO {
	    id: number;
	    commandKind: string;
	    commandSummary: string;
	    createdAt: string;
	    createdBy: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryNodeDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.commandKind = source["commandKind"];
	        this.commandSummary = source["commandSummary"];
	        this.createdAt = source["createdAt"];
	        this.createdBy = source["createdBy"];
	    }
	}
	export class IntegrationProfileDTO {
	    id: number;
	    profileKey: string;
	    sourceChannel: string;
	    sourceSurface: string;
	    demandKind: string;
	    initialAllocationStrategy: string;
	    identityStrategy: string;
	    entitlementAuthorityMode: string;
	    recipientInputMode: string;
	    referenceStrategy: string;
	    trackingSyncMode: string;
	    closurePolicy: string;
	    supportsPartialShipment: boolean;
	    supportsApiImport: boolean;
	    supportsApiExport: boolean;
	    requiresCarrierMapping: boolean;
	    requiresExternalOrderNo: boolean;
	    allowsManualClosure: boolean;
	    connectorKey: string;
	    supportedLocales: string;
	    defaultLocale: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new IntegrationProfileDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.profileKey = source["profileKey"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceSurface = source["sourceSurface"];
	        this.demandKind = source["demandKind"];
	        this.initialAllocationStrategy = source["initialAllocationStrategy"];
	        this.identityStrategy = source["identityStrategy"];
	        this.entitlementAuthorityMode = source["entitlementAuthorityMode"];
	        this.recipientInputMode = source["recipientInputMode"];
	        this.referenceStrategy = source["referenceStrategy"];
	        this.trackingSyncMode = source["trackingSyncMode"];
	        this.closurePolicy = source["closurePolicy"];
	        this.supportsPartialShipment = source["supportsPartialShipment"];
	        this.supportsApiImport = source["supportsApiImport"];
	        this.supportsApiExport = source["supportsApiExport"];
	        this.requiresCarrierMapping = source["requiresCarrierMapping"];
	        this.requiresExternalOrderNo = source["requiresExternalOrderNo"];
	        this.allowsManualClosure = source["allowsManualClosure"];
	        this.connectorKey = source["connectorKey"];
	        this.supportedLocales = source["supportedLocales"];
	        this.defaultLocale = source["defaultLocale"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class IntegrationProfileSummaryDTO {
	    id: number;
	    profileKey: string;
	    sourceChannel: string;
	    trackingSyncMode: string;
	    closurePolicy: string;
	    allowsManualClosure: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IntegrationProfileSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.profileKey = source["profileKey"];
	        this.sourceChannel = source["sourceChannel"];
	        this.trackingSyncMode = source["trackingSyncMode"];
	        this.closurePolicy = source["closurePolicy"];
	        this.allowsManualClosure = source["allowsManualClosure"];
	    }
	}
	export class PlanChannelClosureInput {
	    waveId: number;
	    integrationProfileId: number;
	
	    static createFrom(source: any = {}) {
	        return new PlanChannelClosureInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.integrationProfileId = source["integrationProfileId"];
	    }
	}
	export class PlanChannelClosureResult {
	    decision: string;
	    integrationProfileId: number;
	    trackingSyncMode: string;
	    closurePolicy: string;
	    job?: ChannelSyncJobDTO;
	    items?: ChannelSyncItemDTO[];
	
	    static createFrom(source: any = {}) {
	        return new PlanChannelClosureResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.decision = source["decision"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.trackingSyncMode = source["trackingSyncMode"];
	        this.closurePolicy = source["closurePolicy"];
	        this.job = this.convertValues(source["job"], ChannelSyncJobDTO);
	        this.items = this.convertValues(source["items"], ChannelSyncItemDTO);
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
	export class ProductDTO {
	    id: number;
	    waveId: number;
	    productMasterId?: number;
	    supplierPlatform: string;
	    factorySku: string;
	    name: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ProductDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.productMasterId = source["productMasterId"];
	        this.supplierPlatform = source["supplierPlatform"];
	        this.factorySku = source["factorySku"];
	        this.name = source["name"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProductMasterDTO {
	    id: number;
	    supplierPlatform: string;
	    factorySku: string;
	    supplierProductRef: string;
	    name: string;
	    productKind: string;
	    archived: boolean;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ProductMasterDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.supplierPlatform = source["supplierPlatform"];
	        this.factorySku = source["factorySku"];
	        this.supplierProductRef = source["supplierProductRef"];
	        this.name = source["name"];
	        this.productKind = source["productKind"];
	        this.archived = source["archived"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProfileTemplateBindingDTO {
	    id: number;
	    integrationProfileId: number;
	    documentType: string;
	    templateId: number;
	    isDefault: boolean;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileTemplateBindingDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.documentType = source["documentType"];
	        this.templateId = source["templateId"];
	        this.isDefault = source["isDefault"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ReplayFailureDTO {
	    adjustment_id: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ReplayFailureDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.adjustment_id = source["adjustment_id"];
	        this.reason = source["reason"];
	    }
	}
	export class ReconcileResultDTO {
	    created: number;
	    deleted: number;
	    replayed_count: number;
	    failures: ReplayFailureDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ReconcileResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = source["created"];
	        this.deleted = source["deleted"];
	        this.replayed_count = source["replayed_count"];
	        this.failures = this.convertValues(source["failures"], ReplayFailureDTO);
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
	export class RecordAdjustmentInput {
	    waveId: number;
	    targetKind: string;
	    fulfillmentLineId?: number;
	    waveParticipantSnapshotId?: number;
	    adjustmentKind: string;
	    quantityDelta: number;
	    reasonCode: string;
	    operatorId: string;
	    note: string;
	    evidenceRef: string;
	
	    static createFrom(source: any = {}) {
	        return new RecordAdjustmentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.targetKind = source["targetKind"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.waveParticipantSnapshotId = source["waveParticipantSnapshotId"];
	        this.adjustmentKind = source["adjustmentKind"];
	        this.quantityDelta = source["quantityDelta"];
	        this.reasonCode = source["reasonCode"];
	        this.operatorId = source["operatorId"];
	        this.note = source["note"];
	        this.evidenceRef = source["evidenceRef"];
	    }
	}
	export class RecordClosureDecisionEntry {
	    fulfillmentLineId: number;
	    decisionKind: string;
	    reasonCode: string;
	    note: string;
	    evidenceRef: string;
	    operatorId: string;
	
	    static createFrom(source: any = {}) {
	        return new RecordClosureDecisionEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.decisionKind = source["decisionKind"];
	        this.reasonCode = source["reasonCode"];
	        this.note = source["note"];
	        this.evidenceRef = source["evidenceRef"];
	        this.operatorId = source["operatorId"];
	    }
	}
	export class RecordClosureDecisionInput {
	    waveId: number;
	    integrationProfileId: number;
	    entries: RecordClosureDecisionEntry[];
	
	    static createFrom(source: any = {}) {
	        return new RecordClosureDecisionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.entries = this.convertValues(source["entries"], RecordClosureDecisionEntry);
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
	
	export class ShipmentLineDTO {
	    id: number;
	    shipmentId: number;
	    supplierOrderLineId: number;
	    fulfillmentLineId: number;
	    quantity: number;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ShipmentLineDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.shipmentId = source["shipmentId"];
	        this.supplierOrderLineId = source["supplierOrderLineId"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.quantity = source["quantity"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ShipmentDTO {
	    id: number;
	    supplierOrderId: number;
	    supplierPlatform: string;
	    shipmentNo: string;
	    externalShipmentNo: string;
	    carrierCode: string;
	    carrierName: string;
	    trackingNo: string;
	    status: string;
	    shippedAt: string;
	    basisHistoryNodeId: string;
	    basisProjectionHash: string;
	    basisPayloadSnapshot: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	    lines: ShipmentLineDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ShipmentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.supplierOrderId = source["supplierOrderId"];
	        this.supplierPlatform = source["supplierPlatform"];
	        this.shipmentNo = source["shipmentNo"];
	        this.externalShipmentNo = source["externalShipmentNo"];
	        this.carrierCode = source["carrierCode"];
	        this.carrierName = source["carrierName"];
	        this.trackingNo = source["trackingNo"];
	        this.status = source["status"];
	        this.shippedAt = source["shippedAt"];
	        this.basisHistoryNodeId = source["basisHistoryNodeId"];
	        this.basisProjectionHash = source["basisProjectionHash"];
	        this.basisPayloadSnapshot = source["basisPayloadSnapshot"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.lines = this.convertValues(source["lines"], ShipmentLineDTO);
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
	
	export class SnapshotProductsInput {
	    waveId: number;
	    masterIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new SnapshotProductsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.masterIds = source["masterIds"];
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
	export class SupplierOrderLineDTO {
	    id: number;
	    supplierOrderId: number;
	    fulfillmentLineId: number;
	    supplierLineNo?: number;
	    supplierSku: string;
	    submittedQuantity: number;
	    acceptedQuantity?: number;
	    status: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SupplierOrderLineDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.supplierOrderId = source["supplierOrderId"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.supplierLineNo = source["supplierLineNo"];
	        this.supplierSku = source["supplierSku"];
	        this.submittedQuantity = source["submittedQuantity"];
	        this.acceptedQuantity = source["acceptedQuantity"];
	        this.status = source["status"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class UpdateAllocationPolicyRuleInput {
	    id: number;
	    product_id?: number;
	    selector_payload?: domain.SelectorPayload;
	    product_target_ref?: string;
	    contribution_quantity?: number;
	    rule_kind?: string;
	    priority?: number;
	    active?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateAllocationPolicyRuleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.product_id = source["product_id"];
	        this.selector_payload = this.convertValues(source["selector_payload"], domain.SelectorPayload);
	        this.product_target_ref = source["product_target_ref"];
	        this.contribution_quantity = source["contribution_quantity"];
	        this.rule_kind = source["rule_kind"];
	        this.priority = source["priority"];
	        this.active = source["active"];
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
	export class UpdateProductMasterInput {
	    id: number;
	    supplierPlatform: string;
	    factorySku: string;
	    supplierProductRef: string;
	    name: string;
	    productKind: string;
	    archived: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProductMasterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.supplierPlatform = source["supplierPlatform"];
	        this.factorySku = source["factorySku"];
	        this.supplierProductRef = source["supplierProductRef"];
	        this.name = source["name"];
	        this.productKind = source["productKind"];
	        this.archived = source["archived"];
	    }
	}
	export class UpdateProfileInput {
	    id: number;
	    profileKey: string;
	    sourceChannel: string;
	    sourceSurface: string;
	    demandKind: string;
	    initialAllocationStrategy: string;
	    identityStrategy: string;
	    entitlementAuthorityMode: string;
	    recipientInputMode: string;
	    referenceStrategy: string;
	    trackingSyncMode: string;
	    closurePolicy: string;
	    supportsPartialShipment: boolean;
	    supportsApiImport: boolean;
	    supportsApiExport: boolean;
	    requiresCarrierMapping: boolean;
	    requiresExternalOrderNo: boolean;
	    allowsManualClosure: boolean;
	    connectorKey: string;
	    supportedLocales: string;
	    defaultLocale: string;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProfileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.profileKey = source["profileKey"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceSurface = source["sourceSurface"];
	        this.demandKind = source["demandKind"];
	        this.initialAllocationStrategy = source["initialAllocationStrategy"];
	        this.identityStrategy = source["identityStrategy"];
	        this.entitlementAuthorityMode = source["entitlementAuthorityMode"];
	        this.recipientInputMode = source["recipientInputMode"];
	        this.referenceStrategy = source["referenceStrategy"];
	        this.trackingSyncMode = source["trackingSyncMode"];
	        this.closurePolicy = source["closurePolicy"];
	        this.supportsPartialShipment = source["supportsPartialShipment"];
	        this.supportsApiImport = source["supportsApiImport"];
	        this.supportsApiExport = source["supportsApiExport"];
	        this.requiresCarrierMapping = source["requiresCarrierMapping"];
	        this.requiresExternalOrderNo = source["requiresExternalOrderNo"];
	        this.allowsManualClosure = source["allowsManualClosure"];
	        this.connectorKey = source["connectorKey"];
	        this.supportedLocales = source["supportedLocales"];
	        this.defaultLocale = source["defaultLocale"];
	        this.extraData = source["extraData"];
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
	export class WaveDashboardRowDTO {
	    id: number;
	    waveNo: string;
	    name: string;
	    createdAt: string;
	    projectedLifecycleStage: string;
	
	    static createFrom(source: any = {}) {
	        return new WaveDashboardRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveNo = source["waveNo"];
	        this.name = source["name"];
	        this.createdAt = source["createdAt"];
	        this.projectedLifecycleStage = source["projectedLifecycleStage"];
	    }
	}
	export class WaveFulfillmentRowDTO {
	    fulfillmentLineId: number;
	    waveId: number;
	    waveParticipantSnapshotId?: number;
	    customerProfileId?: number;
	    participantType: string;
	    participantDisplay: string;
	    participantBadge: string;
	    productId?: number;
	    productDisplay: string;
	    demandDocumentId?: number;
	    demandLineId?: number;
	    demandKind: string;
	    demandSourceSummary: string;
	    quantity: number;
	    allocationState: string;
	    addressState: string;
	    supplierState: string;
	    channelSyncState: string;
	    lineReason: string;
	    generatedBy: string;
	    basisDriftStatus: string;
	    reviewRequirement: string;
	    reviewReasonSummary: string;
	
	    static createFrom(source: any = {}) {
	        return new WaveFulfillmentRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.waveId = source["waveId"];
	        this.waveParticipantSnapshotId = source["waveParticipantSnapshotId"];
	        this.customerProfileId = source["customerProfileId"];
	        this.participantType = source["participantType"];
	        this.participantDisplay = source["participantDisplay"];
	        this.participantBadge = source["participantBadge"];
	        this.productId = source["productId"];
	        this.productDisplay = source["productDisplay"];
	        this.demandDocumentId = source["demandDocumentId"];
	        this.demandLineId = source["demandLineId"];
	        this.demandKind = source["demandKind"];
	        this.demandSourceSummary = source["demandSourceSummary"];
	        this.quantity = source["quantity"];
	        this.allocationState = source["allocationState"];
	        this.addressState = source["addressState"];
	        this.supplierState = source["supplierState"];
	        this.channelSyncState = source["channelSyncState"];
	        this.lineReason = source["lineReason"];
	        this.generatedBy = source["generatedBy"];
	        this.basisDriftStatus = source["basisDriftStatus"];
	        this.reviewRequirement = source["reviewRequirement"];
	        this.reviewReasonSummary = source["reviewReasonSummary"];
	    }
	}
	export class WaveOverviewDTO {
	    wave: WaveDTO;
	    demandCount: number;
	    fulfillmentCount: number;
	    supplierOrderCount: number;
	    shipmentCount: number;
	    trackedFulfillmentCount: number;
	    acceptedReadyOrNotRequired: number;
	    acceptedWaitingForInput: number;
	    deferredCount: number;
	    excludedManualCount: number;
	    excludedDuplicateCount: number;
	    excludedRevokedCount: number;
	    mappingBlockedCount: number;
	    channelSyncJobCount: number;
	    channelSyncPendingCount: number;
	    channelSyncRunningCount: number;
	    channelSyncSuccessCount: number;
	    channelSyncPartialSuccessCount: number;
	    channelSyncFailedCount: number;
	    manualClosureDecisionCount: number;
	    manualUnsupportedCount: number;
	    manualSkippedCount: number;
	    manualCompletedCount: number;
	    autoClosureCandidateCount: number;
	    manualClosureCandidateCount: number;
	    projectedLifecycleStage: string;
	    basisDriftSignals: BasisDriftSignalDTO[];
	    hasDriftedBasis: boolean;
	    hasRequiredReviewBasis: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WaveOverviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wave = this.convertValues(source["wave"], WaveDTO);
	        this.demandCount = source["demandCount"];
	        this.fulfillmentCount = source["fulfillmentCount"];
	        this.supplierOrderCount = source["supplierOrderCount"];
	        this.shipmentCount = source["shipmentCount"];
	        this.trackedFulfillmentCount = source["trackedFulfillmentCount"];
	        this.acceptedReadyOrNotRequired = source["acceptedReadyOrNotRequired"];
	        this.acceptedWaitingForInput = source["acceptedWaitingForInput"];
	        this.deferredCount = source["deferredCount"];
	        this.excludedManualCount = source["excludedManualCount"];
	        this.excludedDuplicateCount = source["excludedDuplicateCount"];
	        this.excludedRevokedCount = source["excludedRevokedCount"];
	        this.mappingBlockedCount = source["mappingBlockedCount"];
	        this.channelSyncJobCount = source["channelSyncJobCount"];
	        this.channelSyncPendingCount = source["channelSyncPendingCount"];
	        this.channelSyncRunningCount = source["channelSyncRunningCount"];
	        this.channelSyncSuccessCount = source["channelSyncSuccessCount"];
	        this.channelSyncPartialSuccessCount = source["channelSyncPartialSuccessCount"];
	        this.channelSyncFailedCount = source["channelSyncFailedCount"];
	        this.manualClosureDecisionCount = source["manualClosureDecisionCount"];
	        this.manualUnsupportedCount = source["manualUnsupportedCount"];
	        this.manualSkippedCount = source["manualSkippedCount"];
	        this.manualCompletedCount = source["manualCompletedCount"];
	        this.autoClosureCandidateCount = source["autoClosureCandidateCount"];
	        this.manualClosureCandidateCount = source["manualClosureCandidateCount"];
	        this.projectedLifecycleStage = source["projectedLifecycleStage"];
	        this.basisDriftSignals = this.convertValues(source["basisDriftSignals"], BasisDriftSignalDTO);
	        this.hasDriftedBasis = source["hasDriftedBasis"];
	        this.hasRequiredReviewBasis = source["hasRequiredReviewBasis"];
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
	export class WaveParticipantRowDTO {
	    waveParticipantSnapshotId: number;
	    waveId: number;
	    customerProfileId: number;
	    snapshotType: string;
	    displayName: string;
	    identityPlatform: string;
	    identityValue: string;
	    giftLevel: string;
	    sourceSummary: string;
	    fulfillmentLineCount: number;
	    readyFulfillmentCount: number;
	
	    static createFrom(source: any = {}) {
	        return new WaveParticipantRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveParticipantSnapshotId = source["waveParticipantSnapshotId"];
	        this.waveId = source["waveId"];
	        this.customerProfileId = source["customerProfileId"];
	        this.snapshotType = source["snapshotType"];
	        this.displayName = source["displayName"];
	        this.identityPlatform = source["identityPlatform"];
	        this.identityValue = source["identityValue"];
	        this.giftLevel = source["giftLevel"];
	        this.sourceSummary = source["sourceSummary"];
	        this.fulfillmentLineCount = source["fulfillmentLineCount"];
	        this.readyFulfillmentCount = source["readyFulfillmentCount"];
	    }
	}
	export class WaveStepStateDTO {
	    stepKey: string;
	    status: string;
	    primaryCount: number;
	    secondaryCount: number;
	
	    static createFrom(source: any = {}) {
	        return new WaveStepStateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stepKey = source["stepKey"];
	        this.status = source["status"];
	        this.primaryCount = source["primaryCount"];
	        this.secondaryCount = source["secondaryCount"];
	    }
	}
	export class WaveWorkspaceBasisSummaryDTO {
	    driftedCount: number;
	    requiredReviewCount: number;
	    hasDriftedBasis: boolean;
	    hasRequiredReview: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WaveWorkspaceBasisSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.driftedCount = source["driftedCount"];
	        this.requiredReviewCount = source["requiredReviewCount"];
	        this.hasDriftedBasis = source["hasDriftedBasis"];
	        this.hasRequiredReview = source["hasRequiredReview"];
	    }
	}
	export class WaveWorkspaceGuidanceDTO {
	    code: string;
	    severity: string;
	    targetStepKey: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new WaveWorkspaceGuidanceDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.severity = source["severity"];
	        this.targetStepKey = source["targetStepKey"];
	        this.count = source["count"];
	    }
	}
	export class WaveWorkspaceSnapshotDTO {
	    wave: WaveDTO;
	    projectedLifecycleStage: string;
	    overview: WaveOverviewDTO;
	    stepStates: WaveStepStateDTO[];
	    guidance: WaveWorkspaceGuidanceDTO[];
	    basisSummary: WaveWorkspaceBasisSummaryDTO;
	    recentHistory: HistoryNodeDTO[];
	
	    static createFrom(source: any = {}) {
	        return new WaveWorkspaceSnapshotDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wave = this.convertValues(source["wave"], WaveDTO);
	        this.projectedLifecycleStage = source["projectedLifecycleStage"];
	        this.overview = this.convertValues(source["overview"], WaveOverviewDTO);
	        this.stepStates = this.convertValues(source["stepStates"], WaveStepStateDTO);
	        this.guidance = this.convertValues(source["guidance"], WaveWorkspaceGuidanceDTO);
	        this.basisSummary = this.convertValues(source["basisSummary"], WaveWorkspaceBasisSummaryDTO);
	        this.recentHistory = this.convertValues(source["recentHistory"], HistoryNodeDTO);
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

