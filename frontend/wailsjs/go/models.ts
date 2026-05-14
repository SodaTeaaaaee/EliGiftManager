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
	    basisHistoryNodeId: string;
	    basisProjectionHash: string;
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
	        this.basisHistoryNodeId = source["basisHistoryNodeId"];
	        this.basisProjectionHash = source["basisProjectionHash"];
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
	    shipmentCount: number;
	    trackedFulfillmentCount: number;
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
	    projectedLifecycleStage: string;
	
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
	        this.projectedLifecycleStage = source["projectedLifecycleStage"];
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

}


