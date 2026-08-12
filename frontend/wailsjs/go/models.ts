export namespace dto {
	
	export class ActionCenterBucketFilterDTO {
	    stepKey?: string;
	    allocationState?: string;
	    addressState?: string;
	    supplierState?: string;
	    channelSyncState?: string;
	    reviewRequirement?: string;
	    drift?: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionCenterBucketFilterDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stepKey = source["stepKey"];
	        this.allocationState = source["allocationState"];
	        this.addressState = source["addressState"];
	        this.supplierState = source["supplierState"];
	        this.channelSyncState = source["channelSyncState"];
	        this.reviewRequirement = source["reviewRequirement"];
	        this.drift = source["drift"];
	    }
	}
	export class ActionCenterNavBadgeDTO {
	    navKey: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ActionCenterNavBadgeDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.navKey = source["navKey"];
	        this.count = source["count"];
	    }
	}
	export class ActionCenterWaveBucketDTO {
	    waveId: number;
	    bucketKind: string;
	    count: number;
	    filter: ActionCenterBucketFilterDTO;
	
	    static createFrom(source: any = {}) {
	        return new ActionCenterWaveBucketDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.bucketKind = source["bucketKind"];
	        this.count = source["count"];
	        this.filter = this.convertValues(source["filter"], ActionCenterBucketFilterDTO);
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
	export class ActionCenterWaveSummaryDTO {
	    waveId: number;
	    waveNo: string;
	    waveName: string;
	    buckets: ActionCenterWaveBucketDTO[];
	    totalBlockedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ActionCenterWaveSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.waveNo = source["waveNo"];
	        this.waveName = source["waveName"];
	        this.buckets = this.convertValues(source["buckets"], ActionCenterWaveBucketDTO);
	        this.totalBlockedCount = source["totalBlockedCount"];
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
	export class ActionCenterSummaryDTO {
	    waves: ActionCenterWaveSummaryDTO[];
	    inboxPendingIntakeCount: number;
	    navBadges: ActionCenterNavBadgeDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ActionCenterSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waves = this.convertValues(source["waves"], ActionCenterWaveSummaryDTO);
	        this.inboxPendingIntakeCount = source["inboxPendingIntakeCount"];
	        this.navBadges = this.convertValues(source["navBadges"], ActionCenterNavBadgeDTO);
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
	
	
	export class AddressBatchItemResult {
	    fulfillmentLineId: number;
	    customerAddressId?: number;
	    success: boolean;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new AddressBatchItemResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.customerAddressId = source["customerAddressId"];
	        this.success = source["success"];
	        this.errorMessage = source["errorMessage"];
	    }
	}
	export class AllocationPolicyRuleDTO {
	    id: number;
	    waveId: number;
	    productId: number;
	    selectorPayload: number[];
	    productTargetRef: string;
	    contributionQuantity: number;
	    ruleKind: string;
	    priority: number;
	    active: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AllocationPolicyRuleDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.productId = source["productId"];
	        this.selectorPayload = source["selectorPayload"];
	        this.productTargetRef = source["productTargetRef"];
	        this.contributionQuantity = source["contributionQuantity"];
	        this.ruleKind = source["ruleKind"];
	        this.priority = source["priority"];
	        this.active = source["active"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class FulfillmentAdjustmentDTO {
	    id: number;
	    waveId: number;
	    targetKind: string;
	    fulfillmentLineId?: number;
	    waveParticipantSnapshotId?: number;
	    adjustmentKind: string;
	    quantityDelta: number;
	    fromProductId?: number;
	    toProductId?: number;
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
	        this.fromProductId = source["fromProductId"];
	        this.toProductId = source["toProductId"];
	        this.reasonCode = source["reasonCode"];
	        this.operatorId = source["operatorId"];
	        this.note = source["note"];
	        this.evidenceRef = source["evidenceRef"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class BatchAdjustmentItemResult {
	    index: number;
	    success: boolean;
	    adjustment?: FulfillmentAdjustmentDTO;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchAdjustmentItemResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.success = source["success"];
	        this.adjustment = this.convertValues(source["adjustment"], FulfillmentAdjustmentDTO);
	        this.error = source["error"];
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
	export class BatchAssignDemandInput {
	    waveId: number;
	    docIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new BatchAssignDemandInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.docIds = source["docIds"];
	    }
	}
	export class BatchAssignDemandItemResult {
	    demandDocumentId: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchAssignDemandItemResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demandDocumentId = source["demandDocumentId"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class BatchAssignDemandResult {
	    results: BatchAssignDemandItemResult[];
	    successCount: number;
	    failureCount: number;
	
	    static createFrom(source: any = {}) {
	        return new BatchAssignDemandResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], BatchAssignDemandItemResult);
	        this.successCount = source["successCount"];
	        this.failureCount = source["failureCount"];
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
	    fromProductId?: number;
	    toProductId?: number;
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
	        this.fromProductId = source["fromProductId"];
	        this.toProductId = source["toProductId"];
	        this.reasonCode = source["reasonCode"];
	        this.operatorId = source["operatorId"];
	        this.note = source["note"];
	        this.evidenceRef = source["evidenceRef"];
	    }
	}
	export class BatchRecordAdjustmentsInput {
	    entries: RecordAdjustmentInput[];
	
	    static createFrom(source: any = {}) {
	        return new BatchRecordAdjustmentsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], RecordAdjustmentInput);
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
	export class BatchRecordAdjustmentsResult {
	    results: BatchAdjustmentItemResult[];
	    successCount: number;
	    failureCount: number;
	
	    static createFrom(source: any = {}) {
	        return new BatchRecordAdjustmentsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], BatchAdjustmentItemResult);
	        this.successCount = source["successCount"];
	        this.failureCount = source["failureCount"];
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
	export class UpdateDemandLineRoutingInput {
	    demandLineId: number;
	    routingDisposition: string;
	    recipientInputState: string;
	    routingReasonCode: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateDemandLineRoutingInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demandLineId = source["demandLineId"];
	        this.routingDisposition = source["routingDisposition"];
	        this.recipientInputState = source["recipientInputState"];
	        this.routingReasonCode = source["routingReasonCode"];
	    }
	}
	export class BatchUpdateDemandLineRoutingInput {
	    updates: UpdateDemandLineRoutingInput[];
	
	    static createFrom(source: any = {}) {
	        return new BatchUpdateDemandLineRoutingInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updates = this.convertValues(source["updates"], UpdateDemandLineRoutingInput);
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
	export class DemandLineRoutingError {
	    demandLineId: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new DemandLineRoutingError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demandLineId = source["demandLineId"];
	        this.reason = source["reason"];
	    }
	}
	export class BatchUpdateDemandLineRoutingResult {
	    updatedCount: number;
	    errors: DemandLineRoutingError[];
	
	    static createFrom(source: any = {}) {
	        return new BatchUpdateDemandLineRoutingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updatedCount = source["updatedCount"];
	        this.errors = this.convertValues(source["errors"], DemandLineRoutingError);
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
	export class BindAddressEntry {
	    fulfillmentLineId: number;
	    customerAddressId: number;
	
	    static createFrom(source: any = {}) {
	        return new BindAddressEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.customerAddressId = source["customerAddressId"];
	    }
	}
	export class BindAddressInput {
	    fulfillmentLineId: number;
	    customerAddressId: number;
	
	    static createFrom(source: any = {}) {
	        return new BindAddressInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.customerAddressId = source["customerAddressId"];
	    }
	}
	export class BindInternalCarrierInput {
	    externalCarrierId: number;
	    internalCarrierCode: string;
	
	    static createFrom(source: any = {}) {
	        return new BindInternalCarrierInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.externalCarrierId = source["externalCarrierId"];
	        this.internalCarrierCode = source["internalCarrierCode"];
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
	export class CSVFilePreviewDTO {
	    headers: string[];
	    rows: any[];
	
	    static createFrom(source: any = {}) {
	        return new CSVFilePreviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headers = source["headers"];
	        this.rows = source["rows"];
	    }
	}
	export class CarrierMappingDTO {
	    id: number;
	    integrationProfileId: number;
	    internalCarrierCode: string;
	    externalCarrierCode: string;
	    externalCarrierName: string;
	    aliases: string;
	    isDefault: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CarrierMappingDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.internalCarrierCode = source["internalCarrierCode"];
	        this.externalCarrierCode = source["externalCarrierCode"];
	        this.externalCarrierName = source["externalCarrierName"];
	        this.aliases = source["aliases"];
	        this.isDefault = source["isDefault"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	    startedAt?: string;
	    finishedAt?: string;
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
	export class CloseWaveInput {
	    waveId: number;
	    note: string;
	    force: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CloseWaveInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.note = source["note"];
	        this.force = source["force"];
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
	export class CloseWaveResult {
	    wave: WaveDTO;
	    forced: boolean;
	    residualItemCount: number;
	
	    static createFrom(source: any = {}) {
	        return new CloseWaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wave = this.convertValues(source["wave"], WaveDTO);
	        this.forced = source["forced"];
	        this.residualItemCount = source["residualItemCount"];
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
	export class CreateAddressInput {
	    customerProfileId: number;
	    label: string;
	    recipientName: string;
	    phone: string;
	    country: string;
	    province: string;
	    city: string;
	    district: string;
	    addressLine1: string;
	    addressLine2: string;
	    postalCode: string;
	    isDefault: boolean;
	    isTest: boolean;
	    validationStatus: string;
	    validationDetail: string;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateAddressInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customerProfileId = source["customerProfileId"];
	        this.label = source["label"];
	        this.recipientName = source["recipientName"];
	        this.phone = source["phone"];
	        this.country = source["country"];
	        this.province = source["province"];
	        this.city = source["city"];
	        this.district = source["district"];
	        this.addressLine1 = source["addressLine1"];
	        this.addressLine2 = source["addressLine2"];
	        this.postalCode = source["postalCode"];
	        this.isDefault = source["isDefault"];
	        this.isTest = source["isTest"];
	        this.validationStatus = source["validationStatus"];
	        this.validationDetail = source["validationDetail"];
	        this.extraData = source["extraData"];
	    }
	}
	export class CreateAllocationPolicyRuleInput {
	    waveId: number;
	    productId: number;
	    selectorPayload: number[];
	    productTargetRef: string;
	    contributionQuantity: number;
	    ruleKind: string;
	    priority: number;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateAllocationPolicyRuleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.productId = source["productId"];
	        this.selectorPayload = source["selectorPayload"];
	        this.productTargetRef = source["productTargetRef"];
	        this.contributionQuantity = source["contributionQuantity"];
	        this.ruleKind = source["ruleKind"];
	        this.priority = source["priority"];
	        this.active = source["active"];
	    }
	}
	export class CreateCarrierMappingInput {
	    integrationProfileId: number;
	    internalCarrierCode: string;
	    externalCarrierCode: string;
	    externalCarrierName: string;
	    aliases: string;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateCarrierMappingInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.integrationProfileId = source["integrationProfileId"];
	        this.internalCarrierCode = source["internalCarrierCode"];
	        this.externalCarrierCode = source["externalCarrierCode"];
	        this.externalCarrierName = source["externalCarrierName"];
	        this.aliases = source["aliases"];
	        this.isDefault = source["isDefault"];
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
	export class CreateCustomerIdentityInput {
	    customerProfileId: number;
	    identityPlatform: string;
	    identityValue: string;
	    identityType: string;
	    isPrimary: boolean;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateCustomerIdentityInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customerProfileId = source["customerProfileId"];
	        this.identityPlatform = source["identityPlatform"];
	        this.identityValue = source["identityValue"];
	        this.identityType = source["identityType"];
	        this.isPrimary = source["isPrimary"];
	        this.extraData = source["extraData"];
	    }
	}
	export class CreateCustomerProfileInput {
	    displayName: string;
	    profileType: string;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateCustomerProfileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.profileType = source["profileType"];
	        this.extraData = source["extraData"];
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
	    coverImagePath: string;
	    detailImagePaths: string;
	    extraData: string;
	
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
	        this.coverImagePath = source["coverImagePath"];
	        this.detailImagePaths = source["detailImagePaths"];
	        this.extraData = source["extraData"];
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
	    supportsExportSupplierOrder: boolean;
	    supportsImportProductCatalog: boolean;
	    supportsImportSupplierShipment: boolean;
	    connectorKey: string;
	    factorySupplierPlatform: string;
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
	        this.supportsExportSupplierOrder = source["supportsExportSupplierOrder"];
	        this.supportsImportProductCatalog = source["supportsImportProductCatalog"];
	        this.supportsImportSupplierShipment = source["supportsImportSupplierShipment"];
	        this.connectorKey = source["connectorKey"];
	        this.factorySupplierPlatform = source["factorySupplierPlatform"];
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
	    shippedAt?: string;
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
	    waveType: string;
	    notes: string;
	    levelTags: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateWaveInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.waveType = source["waveType"];
	        this.notes = source["notes"];
	        this.levelTags = source["levelTags"];
	    }
	}
	export class CustomerAddressDTO {
	    id: number;
	    customerProfileId: number;
	    label: string;
	    recipientName: string;
	    phone: string;
	    country: string;
	    province: string;
	    city: string;
	    district: string;
	    addressLine1: string;
	    addressLine2: string;
	    postalCode: string;
	    isDefault: boolean;
	    isTest: boolean;
	    validationStatus: string;
	    validationDetail: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerAddressDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customerProfileId = source["customerProfileId"];
	        this.label = source["label"];
	        this.recipientName = source["recipientName"];
	        this.phone = source["phone"];
	        this.country = source["country"];
	        this.province = source["province"];
	        this.city = source["city"];
	        this.district = source["district"];
	        this.addressLine1 = source["addressLine1"];
	        this.addressLine2 = source["addressLine2"];
	        this.postalCode = source["postalCode"];
	        this.isDefault = source["isDefault"];
	        this.isTest = source["isTest"];
	        this.validationStatus = source["validationStatus"];
	        this.validationDetail = source["validationDetail"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CustomerFulfillmentHistoryRowDTO {
	    fulfillmentLineId: number;
	    waveId: number;
	    waveNo: string;
	    waveName: string;
	    productId?: number;
	    productName: string;
	    productSku: string;
	    quantity: number;
	    allocationState: string;
	    addressState: string;
	    supplierState: string;
	    channelSyncState: string;
	    shipmentId?: number;
	    shipmentStatus: string;
	    trackingNo: string;
	    carrierName: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerFulfillmentHistoryRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.waveId = source["waveId"];
	        this.waveNo = source["waveNo"];
	        this.waveName = source["waveName"];
	        this.productId = source["productId"];
	        this.productName = source["productName"];
	        this.productSku = source["productSku"];
	        this.quantity = source["quantity"];
	        this.allocationState = source["allocationState"];
	        this.addressState = source["addressState"];
	        this.supplierState = source["supplierState"];
	        this.channelSyncState = source["channelSyncState"];
	        this.shipmentId = source["shipmentId"];
	        this.shipmentStatus = source["shipmentStatus"];
	        this.trackingNo = source["trackingNo"];
	        this.carrierName = source["carrierName"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class CustomerIdentityDTO {
	    id: number;
	    customerProfileId: number;
	    identityPlatform: string;
	    identityValue: string;
	    identityType: string;
	    isPrimary: boolean;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerIdentityDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customerProfileId = source["customerProfileId"];
	        this.identityPlatform = source["identityPlatform"];
	        this.identityValue = source["identityValue"];
	        this.identityType = source["identityType"];
	        this.isPrimary = source["isPrimary"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class MergeOperationEventDTO {
	    eventType: string;
	    status: string;
	    actorRef: string;
	    reasonCode: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new MergeOperationEventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.eventType = source["eventType"];
	        this.status = source["status"];
	        this.actorRef = source["actorRef"];
	        this.reasonCode = source["reasonCode"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class MergePlannedEntity {
	    entityType: string;
	    entityId: number;
	    mutationKind: string;
	    fromProfileId: number;
	    toProfileId: number;
	
	    static createFrom(source: any = {}) {
	        return new MergePlannedEntity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entityType = source["entityType"];
	        this.entityId = source["entityId"];
	        this.mutationKind = source["mutationKind"];
	        this.fromProfileId = source["fromProfileId"];
	        this.toProfileId = source["toProfileId"];
	    }
	}
	export class MergeEntityCounts {
	    identities: number;
	    addresses: number;
	    demandDocuments: number;
	    nameObservations: number;
	    nameEvents: number;
	    origins: number;
	    profileMutations: number;
	
	    static createFrom(source: any = {}) {
	        return new MergeEntityCounts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.identities = source["identities"];
	        this.addresses = source["addresses"];
	        this.demandDocuments = source["demandDocuments"];
	        this.nameObservations = source["nameObservations"];
	        this.nameEvents = source["nameEvents"];
	        this.origins = source["origins"];
	        this.profileMutations = source["profileMutations"];
	    }
	}
	export class CustomerMergeHistoryDetail {
	    mergeId: number;
	    sourceProfileId: number;
	    targetProfileId: number;
	    sourceDisplayName: string;
	    targetDisplayName: string;
	    status: string;
	    mergeMode: string;
	    actorRef: string;
	    decisionReason: string;
	    candidateId?: number;
	    policyRevisionId?: number;
	    counts: MergeEntityCounts;
	    auditLevel: string;
	    createdAt: string;
	    undoneAt?: string;
	    canRequestUndoDryRun: boolean;
	    plannedEntities: MergePlannedEntity[];
	    events: MergeOperationEventDTO[];
	    evidenceSnapshot: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergeHistoryDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.sourceDisplayName = source["sourceDisplayName"];
	        this.targetDisplayName = source["targetDisplayName"];
	        this.status = source["status"];
	        this.mergeMode = source["mergeMode"];
	        this.actorRef = source["actorRef"];
	        this.decisionReason = source["decisionReason"];
	        this.candidateId = source["candidateId"];
	        this.policyRevisionId = source["policyRevisionId"];
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.auditLevel = source["auditLevel"];
	        this.createdAt = source["createdAt"];
	        this.undoneAt = source["undoneAt"];
	        this.canRequestUndoDryRun = source["canRequestUndoDryRun"];
	        this.plannedEntities = this.convertValues(source["plannedEntities"], MergePlannedEntity);
	        this.events = this.convertValues(source["events"], MergeOperationEventDTO);
	        this.evidenceSnapshot = source["evidenceSnapshot"];
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
	export class CustomerMergeHistoryItem {
	    mergeId: number;
	    sourceProfileId: number;
	    targetProfileId: number;
	    sourceDisplayName: string;
	    targetDisplayName: string;
	    status: string;
	    mergeMode: string;
	    actorRef: string;
	    decisionReason: string;
	    candidateId?: number;
	    policyRevisionId?: number;
	    counts: MergeEntityCounts;
	    auditLevel: string;
	    createdAt: string;
	    undoneAt?: string;
	    canRequestUndoDryRun: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergeHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.sourceDisplayName = source["sourceDisplayName"];
	        this.targetDisplayName = source["targetDisplayName"];
	        this.status = source["status"];
	        this.mergeMode = source["mergeMode"];
	        this.actorRef = source["actorRef"];
	        this.decisionReason = source["decisionReason"];
	        this.candidateId = source["candidateId"];
	        this.policyRevisionId = source["policyRevisionId"];
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.auditLevel = source["auditLevel"];
	        this.createdAt = source["createdAt"];
	        this.undoneAt = source["undoneAt"];
	        this.canRequestUndoDryRun = source["canRequestUndoDryRun"];
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
	export class CustomerMergeHistoryPage {
	    items: CustomerMergeHistoryItem[];
	    nextCreatedAt?: string;
	    nextId: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergeHistoryPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], CustomerMergeHistoryItem);
	        this.nextCreatedAt = source["nextCreatedAt"];
	        this.nextId = source["nextId"];
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
	export class CustomerMergeHistoryQuery {
	    profileId: number;
	    candidateId: number;
	    status: string;
	    beforeCreatedAt?: string;
	    beforeId: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergeHistoryQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.candidateId = source["candidateId"];
	        this.status = source["status"];
	        this.beforeCreatedAt = source["beforeCreatedAt"];
	        this.beforeId = source["beforeId"];
	        this.limit = source["limit"];
	    }
	}
	export class PrimaryIdentitySelection {
	    namespace: string;
	    identityType: string;
	    identityId: number;
	
	    static createFrom(source: any = {}) {
	        return new PrimaryIdentitySelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.namespace = source["namespace"];
	        this.identityType = source["identityType"];
	        this.identityId = source["identityId"];
	    }
	}
	export class CustomerMergePreviewInput {
	    sourceProfileId: number;
	    targetProfileId: number;
	    candidateId?: number;
	    primaryIdentitySelections: PrimaryIdentitySelection[];
	    defaultAddressId?: number;
	    displayNameResolution: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergePreviewInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.candidateId = source["candidateId"];
	        this.primaryIdentitySelections = this.convertValues(source["primaryIdentitySelections"], PrimaryIdentitySelection);
	        this.defaultAddressId = source["defaultAddressId"];
	        this.displayNameResolution = source["displayNameResolution"];
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
	export class DisplayNameOption {
	    resolution: string;
	    displayName: string;
	    profileId: number;
	
	    static createFrom(source: any = {}) {
	        return new DisplayNameOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolution = source["resolution"];
	        this.displayName = source["displayName"];
	        this.profileId = source["profileId"];
	    }
	}
	export class DefaultAddressOption {
	    addressId: number;
	    customerProfileId: number;
	    displayValue: string;
	    currentDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DefaultAddressOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.addressId = source["addressId"];
	        this.customerProfileId = source["customerProfileId"];
	        this.displayValue = source["displayValue"];
	        this.currentDefault = source["currentDefault"];
	    }
	}
	export class PrimaryIdentityOption {
	    namespace: string;
	    identityType: string;
	    identityId: number;
	    customerProfileId: number;
	    displayValue: string;
	    currentPrimary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PrimaryIdentityOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.namespace = source["namespace"];
	        this.identityType = source["identityType"];
	        this.identityId = source["identityId"];
	        this.customerProfileId = source["customerProfileId"];
	        this.displayValue = source["displayValue"];
	        this.currentPrimary = source["currentPrimary"];
	    }
	}
	export class MergeBlocker {
	    code: string;
	    entityType: string;
	    entityId: number;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new MergeBlocker(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.entityType = source["entityType"];
	        this.entityId = source["entityId"];
	        this.detail = source["detail"];
	    }
	}
	export class CustomerMergePreviewResult {
	    previewToken: string;
	    previewHash: string;
	    generatedAt: string;
	    sourceProfileId: number;
	    targetProfileId: number;
	    sourceStatus: string;
	    targetStatus: string;
	    sourceRowVersion: number;
	    targetRowVersion: number;
	    candidateId?: number;
	    candidateRowVersion: number;
	    evidenceHash: string;
	    policyVersion: number;
	    policyRevisionId?: number;
	    dependsOnMergeRecordId?: number;
	    plannedEntities: MergePlannedEntity[];
	    frozenDemandDocumentIds: number[];
	    counts: MergeEntityCounts;
	    blockers: MergeBlocker[];
	    canExecute: boolean;
	    primaryIdentityOptions: PrimaryIdentityOption[];
	    recommendedPrimaryIdentitySelections: PrimaryIdentitySelection[];
	    defaultAddressOptions: DefaultAddressOption[];
	    recommendedDefaultAddressId?: number;
	    displayNameOptions: DisplayNameOption[];
	    recommendedDisplayNameResolution: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergePreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.previewToken = source["previewToken"];
	        this.previewHash = source["previewHash"];
	        this.generatedAt = source["generatedAt"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.sourceStatus = source["sourceStatus"];
	        this.targetStatus = source["targetStatus"];
	        this.sourceRowVersion = source["sourceRowVersion"];
	        this.targetRowVersion = source["targetRowVersion"];
	        this.candidateId = source["candidateId"];
	        this.candidateRowVersion = source["candidateRowVersion"];
	        this.evidenceHash = source["evidenceHash"];
	        this.policyVersion = source["policyVersion"];
	        this.policyRevisionId = source["policyRevisionId"];
	        this.dependsOnMergeRecordId = source["dependsOnMergeRecordId"];
	        this.plannedEntities = this.convertValues(source["plannedEntities"], MergePlannedEntity);
	        this.frozenDemandDocumentIds = source["frozenDemandDocumentIds"];
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.blockers = this.convertValues(source["blockers"], MergeBlocker);
	        this.canExecute = source["canExecute"];
	        this.primaryIdentityOptions = this.convertValues(source["primaryIdentityOptions"], PrimaryIdentityOption);
	        this.recommendedPrimaryIdentitySelections = this.convertValues(source["recommendedPrimaryIdentitySelections"], PrimaryIdentitySelection);
	        this.defaultAddressOptions = this.convertValues(source["defaultAddressOptions"], DefaultAddressOption);
	        this.recommendedDefaultAddressId = source["recommendedDefaultAddressId"];
	        this.displayNameOptions = this.convertValues(source["displayNameOptions"], DisplayNameOption);
	        this.recommendedDisplayNameResolution = source["recommendedDisplayNameResolution"];
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
	export class CustomerMergeUndoDryRunInput {
	    mergeId: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergeUndoDryRunInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	    }
	}
	export class CustomerMergeUndoDryRunResult {
	    mergeId: number;
	    eligible: boolean;
	    eligibilityToken: string;
	    generatedAt: string;
	    sourceRowVersion: number;
	    targetRowVersion: number;
	    restoreCounts: MergeEntityCounts;
	    blockers: MergeBlocker[];
	    warnings: string[];
	    auditLevel: string;
	    dependentMergeIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new CustomerMergeUndoDryRunResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	        this.eligible = source["eligible"];
	        this.eligibilityToken = source["eligibilityToken"];
	        this.generatedAt = source["generatedAt"];
	        this.sourceRowVersion = source["sourceRowVersion"];
	        this.targetRowVersion = source["targetRowVersion"];
	        this.restoreCounts = this.convertValues(source["restoreCounts"], MergeEntityCounts);
	        this.blockers = this.convertValues(source["blockers"], MergeBlocker);
	        this.warnings = source["warnings"];
	        this.auditLevel = source["auditLevel"];
	        this.dependentMergeIds = source["dependentMergeIds"];
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
	export class CustomerNameObservationDTO {
	    id: number;
	    kind: string;
	    value: string;
	    source: string;
	    firstSeenAt?: string;
	    lastSeenAt?: string;
	    count: number;
	    isDisplayNameSource: boolean;
	    originProfileId: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerNameObservationDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.source = source["source"];
	        this.firstSeenAt = source["firstSeenAt"];
	        this.lastSeenAt = source["lastSeenAt"];
	        this.count = source["count"];
	        this.isDisplayNameSource = source["isDisplayNameSource"];
	        this.originProfileId = source["originProfileId"];
	    }
	}
	export class CustomerProfileDTO {
	    id: number;
	    displayName: string;
	    profileType: string;
	    status: string;
	    mergedIntoProfileId?: number;
	    rowVersion: number;
	    displayNameMode: string;
	    displayNameObservationId?: number;
	    matchedHistoricalName?: string;
	    extraData: string;
	    createdAt: string;
	    updatedAt: string;
	    identities: CustomerIdentityDTO[];
	    addresses: CustomerAddressDTO[];
	    activeAddressCount: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerProfileDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.profileType = source["profileType"];
	        this.status = source["status"];
	        this.mergedIntoProfileId = source["mergedIntoProfileId"];
	        this.rowVersion = source["rowVersion"];
	        this.displayNameMode = source["displayNameMode"];
	        this.displayNameObservationId = source["displayNameObservationId"];
	        this.matchedHistoricalName = source["matchedHistoricalName"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.identities = this.convertValues(source["identities"], CustomerIdentityDTO);
	        this.addresses = this.convertValues(source["addresses"], CustomerAddressDTO);
	        this.activeAddressCount = source["activeAddressCount"];
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
	export class CustomerProfileOriginDTO {
	    id: number;
	    customerProfileId: number;
	    originKind: string;
	    sourceIntegrationProfileId?: number;
	    externalRef: string;
	    sourceDocumentId?: number;
	    lastSeenAt?: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerProfileOriginDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customerProfileId = source["customerProfileId"];
	        this.originKind = source["originKind"];
	        this.sourceIntegrationProfileId = source["sourceIntegrationProfileId"];
	        this.externalRef = source["externalRef"];
	        this.sourceDocumentId = source["sourceDocumentId"];
	        this.lastSeenAt = source["lastSeenAt"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class CustomerProfilePageFilterInput {
	    keyword: string;
	    platform: string;
	    missingAddressOnly: boolean;
	    sortBy: string;
	    sortDir: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerProfilePageFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.platform = source["platform"];
	        this.missingAddressOnly = source["missingAddressOnly"];
	        this.sortBy = source["sortBy"];
	        this.sortDir = source["sortDir"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class CustomerProfilePageResult {
	    items: CustomerProfileDTO[];
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerProfilePageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], CustomerProfileDTO);
	        this.totalCount = source["totalCount"];
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
	export class CustomerResolutionFeaturePolicyDTO {
	    revision: number;
	    customerResolutionWritesEnabled: boolean;
	    candidateScanEnabled: boolean;
	    mergeExecutionEnabled: boolean;
	    splitExecutionEnabled: boolean;
	    importEvidenceEnabled: boolean;
	    carrierRegistryWritesEnabled: boolean;
	    actorRef: string;
	    reason: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerResolutionFeaturePolicyDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.customerResolutionWritesEnabled = source["customerResolutionWritesEnabled"];
	        this.candidateScanEnabled = source["candidateScanEnabled"];
	        this.mergeExecutionEnabled = source["mergeExecutionEnabled"];
	        this.splitExecutionEnabled = source["splitExecutionEnabled"];
	        this.importEvidenceEnabled = source["importEvidenceEnabled"];
	        this.carrierRegistryWritesEnabled = source["carrierRegistryWritesEnabled"];
	        this.actorRef = source["actorRef"];
	        this.reason = source["reason"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CustomerSplitOperationEventDTO {
	    eventType: string;
	    status: string;
	    actorRef: string;
	    reasonCode: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitOperationEventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.eventType = source["eventType"];
	        this.status = source["status"];
	        this.actorRef = source["actorRef"];
	        this.reasonCode = source["reasonCode"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class CustomerSplitMovedEntityDTO {
	    entityType: string;
	    entityId: number;
	    fromProfileId: number;
	    toProfileId: number;
	    mutationKind: string;
	    beforeSnapshot: string;
	    afterSnapshot: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitMovedEntityDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entityType = source["entityType"];
	        this.entityId = source["entityId"];
	        this.fromProfileId = source["fromProfileId"];
	        this.toProfileId = source["toProfileId"];
	        this.mutationKind = source["mutationKind"];
	        this.beforeSnapshot = source["beforeSnapshot"];
	        this.afterSnapshot = source["afterSnapshot"];
	    }
	}
	export class CustomerSplitHistoryDetail {
	    splitId: number;
	    operationType: string;
	    sourceProfileId: number;
	    targetProfileId: number;
	    targetStrategy: string;
	    status: string;
	    actorRef: string;
	    decisionReason: string;
	    counts: MergeEntityCounts;
	    createdAt: string;
	    completedAt?: string;
	    directUndoSupported: boolean;
	    reverseOperationKind: string;
	    movedEntities: CustomerSplitMovedEntityDTO[];
	    events: CustomerSplitOperationEventDTO[];
	    planHash: string;
	    reverseGuidance: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitHistoryDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.splitId = source["splitId"];
	        this.operationType = source["operationType"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.targetStrategy = source["targetStrategy"];
	        this.status = source["status"];
	        this.actorRef = source["actorRef"];
	        this.decisionReason = source["decisionReason"];
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.createdAt = source["createdAt"];
	        this.completedAt = source["completedAt"];
	        this.directUndoSupported = source["directUndoSupported"];
	        this.reverseOperationKind = source["reverseOperationKind"];
	        this.movedEntities = this.convertValues(source["movedEntities"], CustomerSplitMovedEntityDTO);
	        this.events = this.convertValues(source["events"], CustomerSplitOperationEventDTO);
	        this.planHash = source["planHash"];
	        this.reverseGuidance = source["reverseGuidance"];
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
	export class CustomerSplitHistoryItem {
	    splitId: number;
	    operationType: string;
	    sourceProfileId: number;
	    targetProfileId: number;
	    targetStrategy: string;
	    status: string;
	    actorRef: string;
	    decisionReason: string;
	    counts: MergeEntityCounts;
	    createdAt: string;
	    completedAt?: string;
	    directUndoSupported: boolean;
	    reverseOperationKind: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.splitId = source["splitId"];
	        this.operationType = source["operationType"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.targetStrategy = source["targetStrategy"];
	        this.status = source["status"];
	        this.actorRef = source["actorRef"];
	        this.decisionReason = source["decisionReason"];
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.createdAt = source["createdAt"];
	        this.completedAt = source["completedAt"];
	        this.directUndoSupported = source["directUndoSupported"];
	        this.reverseOperationKind = source["reverseOperationKind"];
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
	export class CustomerSplitHistoryPage {
	    items: CustomerSplitHistoryItem[];
	    hasMore: boolean;
	    nextBefore?: string;
	    nextId: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitHistoryPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], CustomerSplitHistoryItem);
	        this.hasMore = source["hasMore"];
	        this.nextBefore = source["nextBefore"];
	        this.nextId = source["nextId"];
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
	export class CustomerSplitHistoryQuery {
	    profileId: number;
	    status: string;
	    beforeCreatedAt?: string;
	    beforeId: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitHistoryQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.status = source["status"];
	        this.beforeCreatedAt = source["beforeCreatedAt"];
	        this.beforeId = source["beforeId"];
	        this.limit = source["limit"];
	    }
	}
	
	
	export class CustomerSplitSelection {
	    identityIds: number[];
	    addressIds: number[];
	    demandDocumentIds: number[];
	    nameObservationIds: number[];
	    originIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.identityIds = source["identityIds"];
	        this.addressIds = source["addressIds"];
	        this.demandDocumentIds = source["demandDocumentIds"];
	        this.nameObservationIds = source["nameObservationIds"];
	        this.originIds = source["originIds"];
	    }
	}
	export class CustomerSplitPreviewInput {
	    sourceProfileId: number;
	    targetStrategy: string;
	    newProfileDisplayName: string;
	    newProfileType: string;
	    targetPrimaryIdentityIds: number[];
	    targetDefaultAddressId?: number;
	    targetDisplayNameObservationId?: number;
	    sourceDisplayNameResolution: string;
	    selection: CustomerSplitSelection;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitPreviewInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetStrategy = source["targetStrategy"];
	        this.newProfileDisplayName = source["newProfileDisplayName"];
	        this.newProfileType = source["newProfileType"];
	        this.targetPrimaryIdentityIds = source["targetPrimaryIdentityIds"];
	        this.targetDefaultAddressId = source["targetDefaultAddressId"];
	        this.targetDisplayNameObservationId = source["targetDisplayNameObservationId"];
	        this.sourceDisplayNameResolution = source["sourceDisplayNameResolution"];
	        this.selection = this.convertValues(source["selection"], CustomerSplitSelection);
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
	export class SplitImmutableHistoryDTO {
	    waveParticipantSnapshotIds: number[];
	    fulfillmentLineIds: number[];
	    willRewrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SplitImmutableHistoryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveParticipantSnapshotIds = source["waveParticipantSnapshotIds"];
	        this.fulfillmentLineIds = source["fulfillmentLineIds"];
	        this.willRewrite = source["willRewrite"];
	    }
	}
	export class CustomerSplitPreviewResult {
	    planToken: string;
	    planHash: string;
	    generatedAt: string;
	    sourceProfileId: number;
	    targetProfileId: number;
	    targetStrategy: string;
	    sourceRowVersion: number;
	    targetRowVersion: number;
	    sourceDisplayNameAfter: string;
	    targetDisplayNameAfter: string;
	    plannedEntities: MergePlannedEntity[];
	    counts: MergeEntityCounts;
	    blockers: MergeBlocker[];
	    immutableHistory: SplitImmutableHistoryDTO;
	    canExecute: boolean;
	    directUndoSupported: boolean;
	    reverseOperationKind: string;
	    unsupportedTargetStrategyHint: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerSplitPreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.planToken = source["planToken"];
	        this.planHash = source["planHash"];
	        this.generatedAt = source["generatedAt"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.targetStrategy = source["targetStrategy"];
	        this.sourceRowVersion = source["sourceRowVersion"];
	        this.targetRowVersion = source["targetRowVersion"];
	        this.sourceDisplayNameAfter = source["sourceDisplayNameAfter"];
	        this.targetDisplayNameAfter = source["targetDisplayNameAfter"];
	        this.plannedEntities = this.convertValues(source["plannedEntities"], MergePlannedEntity);
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.blockers = this.convertValues(source["blockers"], MergeBlocker);
	        this.immutableHistory = this.convertValues(source["immutableHistory"], SplitImmutableHistoryDTO);
	        this.canExecute = source["canExecute"];
	        this.directUndoSupported = source["directUndoSupported"];
	        this.reverseOperationKind = source["reverseOperationKind"];
	        this.unsupportedTargetStrategyHint = source["unsupportedTargetStrategyHint"];
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
	
	
	export class DemandCSVImportError {
	    rowIndex: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new DemandCSVImportError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rowIndex = source["rowIndex"];
	        this.reason = source["reason"];
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
	    sourceCreatedAt?: string;
	    sourcePaidAt?: string;
	    currency: string;
	    authoritySnapshotAt?: string;
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
	    integrationProfileId?: number;
	    waveId?: number;
	    sortBy: string;
	    sortDir: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new DemandInboxFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assignment = source["assignment"];
	        this.demandKind = source["demandKind"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.waveId = source["waveId"];
	        this.sortBy = source["sortBy"];
	        this.sortDir = source["sortDir"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
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
	export class DemandInboxPageResult {
	    items: DemandInboxRowDTO[];
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new DemandInboxPageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], DemandInboxRowDTO);
	        this.totalCount = source["totalCount"];
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
	
	export class PaginationResult {
	    page: number;
	    pageSize: number;
	    totalCount: number;
	    totalPages: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.totalCount = source["totalCount"];
	        this.totalPages = source["totalPages"];
	    }
	}
	export class DemandInboxRowListDTO {
	    rows: DemandInboxRowDTO[];
	    pagination: PaginationResult;
	
	    static createFrom(source: any = {}) {
	        return new DemandInboxRowListDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], DemandInboxRowDTO);
	        this.pagination = this.convertValues(source["pagination"], PaginationResult);
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
	export class DismissMergeCandidateInput {
	    id: number;
	    evidenceHash: string;
	    policyVersion: number;
	
	    static createFrom(source: any = {}) {
	        return new DismissMergeCandidateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.evidenceHash = source["evidenceHash"];
	        this.policyVersion = source["policyVersion"];
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
	export class ExecuteCustomerMergeInput {
	    operationKey: string;
	    previewToken: string;
	    sourceProfileId: number;
	    targetProfileId: number;
	    expectedSourceRowVersion: number;
	    expectedTargetRowVersion: number;
	    candidateId?: number;
	    expectedCandidateRowVersion: number;
	    expectedEvidenceHash: string;
	    expectedPolicyVersion: number;
	    expectedPolicyRevisionId?: number;
	    primaryIdentitySelections: PrimaryIdentitySelection[];
	    defaultAddressId?: number;
	    displayNameResolution: string;
	    actorRef: string;
	    decisionReason: string;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteCustomerMergeInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationKey = source["operationKey"];
	        this.previewToken = source["previewToken"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.expectedSourceRowVersion = source["expectedSourceRowVersion"];
	        this.expectedTargetRowVersion = source["expectedTargetRowVersion"];
	        this.candidateId = source["candidateId"];
	        this.expectedCandidateRowVersion = source["expectedCandidateRowVersion"];
	        this.expectedEvidenceHash = source["expectedEvidenceHash"];
	        this.expectedPolicyVersion = source["expectedPolicyVersion"];
	        this.expectedPolicyRevisionId = source["expectedPolicyRevisionId"];
	        this.primaryIdentitySelections = this.convertValues(source["primaryIdentitySelections"], PrimaryIdentitySelection);
	        this.defaultAddressId = source["defaultAddressId"];
	        this.displayNameResolution = source["displayNameResolution"];
	        this.actorRef = source["actorRef"];
	        this.decisionReason = source["decisionReason"];
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
	export class ExecuteCustomerMergeResult {
	    mergeId: number;
	    operationKey: string;
	    status: string;
	    counts: MergeEntityCounts;
	    sourceRowVersion: number;
	    targetRowVersion: number;
	    candidateStatus: string;
	    undoDryRunRequired: boolean;
	    idempotentReplay: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteCustomerMergeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	        this.operationKey = source["operationKey"];
	        this.status = source["status"];
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.sourceRowVersion = source["sourceRowVersion"];
	        this.targetRowVersion = source["targetRowVersion"];
	        this.candidateStatus = source["candidateStatus"];
	        this.undoDryRunRequired = source["undoDryRunRequired"];
	        this.idempotentReplay = source["idempotentReplay"];
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
	export class ExecuteCustomerMergeUndoInput {
	    mergeId: number;
	    undoOperationKey: string;
	    eligibilityToken: string;
	    expectedSourceRowVersion: number;
	    expectedTargetRowVersion: number;
	    actorRef: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteCustomerMergeUndoInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	        this.undoOperationKey = source["undoOperationKey"];
	        this.eligibilityToken = source["eligibilityToken"];
	        this.expectedSourceRowVersion = source["expectedSourceRowVersion"];
	        this.expectedTargetRowVersion = source["expectedTargetRowVersion"];
	        this.actorRef = source["actorRef"];
	        this.reason = source["reason"];
	    }
	}
	export class ExecuteCustomerMergeUndoResult {
	    mergeId: number;
	    status: string;
	    restoredSourceProfileId: number;
	    targetProfileId: number;
	    restoreCounts: MergeEntityCounts;
	    idempotentReplay: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteCustomerMergeUndoResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	        this.status = source["status"];
	        this.restoredSourceProfileId = source["restoredSourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.restoreCounts = this.convertValues(source["restoreCounts"], MergeEntityCounts);
	        this.idempotentReplay = source["idempotentReplay"];
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
	export class ExecuteCustomerSplitInput {
	    operationKey: string;
	    planToken: string;
	    expectedSourceRowVersion: number;
	    expectedTargetRowVersion: number;
	    actorRef: string;
	    decisionReason: string;
	    plan: CustomerSplitPreviewInput;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteCustomerSplitInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationKey = source["operationKey"];
	        this.planToken = source["planToken"];
	        this.expectedSourceRowVersion = source["expectedSourceRowVersion"];
	        this.expectedTargetRowVersion = source["expectedTargetRowVersion"];
	        this.actorRef = source["actorRef"];
	        this.decisionReason = source["decisionReason"];
	        this.plan = this.convertValues(source["plan"], CustomerSplitPreviewInput);
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
	export class ExecuteCustomerSplitResult {
	    splitId: number;
	    operationKey: string;
	    status: string;
	    sourceProfileId: number;
	    targetProfileId: number;
	    counts: MergeEntityCounts;
	    sourceRowVersion: number;
	    targetRowVersion: number;
	    idempotentReplay: boolean;
	    directUndoSupported: boolean;
	    reverseOperationKind: string;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteCustomerSplitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.splitId = source["splitId"];
	        this.operationKey = source["operationKey"];
	        this.status = source["status"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.counts = this.convertValues(source["counts"], MergeEntityCounts);
	        this.sourceRowVersion = source["sourceRowVersion"];
	        this.targetRowVersion = source["targetRowVersion"];
	        this.idempotentReplay = source["idempotentReplay"];
	        this.directUndoSupported = source["directUndoSupported"];
	        this.reverseOperationKind = source["reverseOperationKind"];
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
	    startedAt?: string;
	    finishedAt?: string;
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
	export class ExternalCarrierDTO {
	    id: number;
	    integrationProfileId: number;
	    canonicalKey: string;
	    externalCarrierCode: string;
	    externalCarrierName: string;
	    nameKeyStrategy: string;
	    internalCarrierCode?: string;
	    status: string;
	    conflictReason: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ExternalCarrierDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.canonicalKey = source["canonicalKey"];
	        this.externalCarrierCode = source["externalCarrierCode"];
	        this.externalCarrierName = source["externalCarrierName"];
	        this.nameKeyStrategy = source["nameKeyStrategy"];
	        this.internalCarrierCode = source["internalCarrierCode"];
	        this.status = source["status"];
	        this.conflictReason = source["conflictReason"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	
	
	export class HistoryGraphNodeDTO {
	    id: number;
	    parentNodeId: number;
	    preferredRedoChildId: number;
	    commandKind: string;
	    commandSummary: string;
	    projectionHash: string;
	    checkpointHint: boolean;
	    createdAt: string;
	    createdBy: string;
	    isCurrentHead: boolean;
	    isPinned: boolean;
	    childCount: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryGraphNodeDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentNodeId = source["parentNodeId"];
	        this.preferredRedoChildId = source["preferredRedoChildId"];
	        this.commandKind = source["commandKind"];
	        this.commandSummary = source["commandSummary"];
	        this.projectionHash = source["projectionHash"];
	        this.checkpointHint = source["checkpointHint"];
	        this.createdAt = source["createdAt"];
	        this.createdBy = source["createdBy"];
	        this.isCurrentHead = source["isCurrentHead"];
	        this.isPinned = source["isPinned"];
	        this.childCount = source["childCount"];
	    }
	}
	export class HistoryGraphDTO {
	    scopeId: number;
	    currentHeadId: number;
	    nodes: HistoryGraphNodeDTO[];
	
	    static createFrom(source: any = {}) {
	        return new HistoryGraphDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scopeId = source["scopeId"];
	        this.currentHeadId = source["currentHeadId"];
	        this.nodes = this.convertValues(source["nodes"], HistoryGraphNodeDTO);
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
	
	export class HistoryNodeDTO {
	    id: number;
	    parentNodeId: number;
	    preferredRedoChildId: number;
	    commandKind: string;
	    commandSummary: string;
	    projectionHash: string;
	    checkpointHint: boolean;
	    createdAt: string;
	    createdBy: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryNodeDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentNodeId = source["parentNodeId"];
	        this.preferredRedoChildId = source["preferredRedoChildId"];
	        this.commandKind = source["commandKind"];
	        this.commandSummary = source["commandSummary"];
	        this.projectionHash = source["projectionHash"];
	        this.checkpointHint = source["checkpointHint"];
	        this.createdAt = source["createdAt"];
	        this.createdBy = source["createdBy"];
	    }
	}
	export class ImportCarrierMappingError {
	    rowIndex: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportCarrierMappingError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rowIndex = source["rowIndex"];
	        this.reason = source["reason"];
	    }
	}
	export class ImportCarrierMappingsInput {
	    integrationProfileId: number;
	    importMode: string;
	    filePath: string;
	    rows: any[];
	
	    static createFrom(source: any = {}) {
	        return new ImportCarrierMappingsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.integrationProfileId = source["integrationProfileId"];
	        this.importMode = source["importMode"];
	        this.filePath = source["filePath"];
	        this.rows = source["rows"];
	    }
	}
	export class ImportCarrierMappingsResult {
	    importRunId: number;
	    evidenceDisabled: boolean;
	    createdCount: number;
	    updatedCount: number;
	    totalProcessed: number;
	    successCount: number;
	    errorCount: number;
	    errors: ImportCarrierMappingError[];
	    mappings: CarrierMappingDTO[];
	    externalCarriers: ExternalCarrierDTO[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportCarrierMappingsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importRunId = source["importRunId"];
	        this.evidenceDisabled = source["evidenceDisabled"];
	        this.createdCount = source["createdCount"];
	        this.updatedCount = source["updatedCount"];
	        this.totalProcessed = source["totalProcessed"];
	        this.successCount = source["successCount"];
	        this.errorCount = source["errorCount"];
	        this.errors = this.convertValues(source["errors"], ImportCarrierMappingError);
	        this.mappings = this.convertValues(source["mappings"], CarrierMappingDTO);
	        this.externalCarriers = this.convertValues(source["externalCarriers"], ExternalCarrierDTO);
	        this.warnings = source["warnings"];
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
	export class ImportDemandCSVInput {
	    integrationProfileId: number;
	    documentType: string;
	    sourceDocumentNo: string;
	    sourceCustomerRef: string;
	    importMode: string;
	    mappingRules: string;
	    rows: any[];
	    filePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportDemandCSVInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.integrationProfileId = source["integrationProfileId"];
	        this.documentType = source["documentType"];
	        this.sourceDocumentNo = source["sourceDocumentNo"];
	        this.sourceCustomerRef = source["sourceCustomerRef"];
	        this.importMode = source["importMode"];
	        this.mappingRules = source["mappingRules"];
	        this.rows = source["rows"];
	        this.filePath = source["filePath"];
	    }
	}
	export class ImportDemandCSVResult {
	    importRunId: number;
	    evidenceDisabled: boolean;
	    document?: DemandDocumentDTO;
	    errors: DemandCSVImportError[];
	    totalProcessed: number;
	    successCount: number;
	    errorCount: number;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportDemandCSVResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importRunId = source["importRunId"];
	        this.evidenceDisabled = source["evidenceDisabled"];
	        this.document = this.convertValues(source["document"], DemandDocumentDTO);
	        this.errors = this.convertValues(source["errors"], DemandCSVImportError);
	        this.totalProcessed = source["totalProcessed"];
	        this.successCount = source["successCount"];
	        this.errorCount = source["errorCount"];
	        this.warnings = source["warnings"];
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
	export class ImportDemandTemplateInput {
	    integrationProfileId: number;
	    documentType: string;
	    sourceDocumentNo: string;
	    sourceCustomerRef: string;
	    rows: any[];
	
	    static createFrom(source: any = {}) {
	        return new ImportDemandTemplateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.integrationProfileId = source["integrationProfileId"];
	        this.documentType = source["documentType"];
	        this.sourceDocumentNo = source["sourceDocumentNo"];
	        this.sourceCustomerRef = source["sourceCustomerRef"];
	        this.rows = source["rows"];
	    }
	}
	export class ImportEvidenceRetentionDTO {
	    retentionDays: number;
	    revision: number;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportEvidenceRetentionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.retentionDays = source["retentionDays"];
	        this.revision = source["revision"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ImportProductCatalogError {
	    rowIndex: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportProductCatalogError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rowIndex = source["rowIndex"];
	        this.reason = source["reason"];
	    }
	}
	export class ImportProductCatalogInput {
	    integrationProfileId: number;
	    importMode: string;
	    filePath: string;
	    rows: any[];
	
	    static createFrom(source: any = {}) {
	        return new ImportProductCatalogInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.integrationProfileId = source["integrationProfileId"];
	        this.importMode = source["importMode"];
	        this.filePath = source["filePath"];
	        this.rows = source["rows"];
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
	    coverImagePath: string;
	    detailImagePaths: string;
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
	        this.coverImagePath = source["coverImagePath"];
	        this.detailImagePaths = source["detailImagePaths"];
	        this.extraData = source["extraData"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ImportProductCatalogResult {
	    importRunId: number;
	    evidenceDisabled: boolean;
	    createdCount: number;
	    updatedCount: number;
	    totalProcessed: number;
	    successCount: number;
	    errorCount: number;
	    errors: ImportProductCatalogError[];
	    masters: ProductMasterDTO[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportProductCatalogResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importRunId = source["importRunId"];
	        this.evidenceDisabled = source["evidenceDisabled"];
	        this.createdCount = source["createdCount"];
	        this.updatedCount = source["updatedCount"];
	        this.totalProcessed = source["totalProcessed"];
	        this.successCount = source["successCount"];
	        this.errorCount = source["errorCount"];
	        this.errors = this.convertValues(source["errors"], ImportProductCatalogError);
	        this.masters = this.convertValues(source["masters"], ProductMasterDTO);
	        this.warnings = source["warnings"];
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
	export class ImportRawRecordDetailDTO {
	    id: number;
	    rowIndex: number;
	    rawLogicalRow: string;
	    unmappedSource: string;
	    parserMetadata: string;
	    warningCodes: string;
	    assetMembers: string;
	    outcome: string;
	    errorCode: string;
	    errorMessage: string;
	    resultType: string;
	    resultId?: number;
	    expiresAt?: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportRawRecordDetailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rowIndex = source["rowIndex"];
	        this.rawLogicalRow = source["rawLogicalRow"];
	        this.unmappedSource = source["unmappedSource"];
	        this.parserMetadata = source["parserMetadata"];
	        this.warningCodes = source["warningCodes"];
	        this.assetMembers = source["assetMembers"];
	        this.outcome = source["outcome"];
	        this.errorCode = source["errorCode"];
	        this.errorMessage = source["errorMessage"];
	        this.resultType = source["resultType"];
	        this.resultId = source["resultId"];
	        this.expiresAt = source["expiresAt"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ImportRunSummaryDTO {
	    id: number;
	    importKind: string;
	    integrationProfileId?: number;
	    sourceFormat: string;
	    sourceFileName: string;
	    importMode: string;
	    status: string;
	    retentionDays: number;
	    retentionPolicyVersion: number;
	    expiresAt?: string;
	    recordCount: number;
	    successCount: number;
	    failureCount: number;
	    quarantinedCount: number;
	    createdAt: string;
	    completedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportRunSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.importKind = source["importKind"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.sourceFormat = source["sourceFormat"];
	        this.sourceFileName = source["sourceFileName"];
	        this.importMode = source["importMode"];
	        this.status = source["status"];
	        this.retentionDays = source["retentionDays"];
	        this.retentionPolicyVersion = source["retentionPolicyVersion"];
	        this.expiresAt = source["expiresAt"];
	        this.recordCount = source["recordCount"];
	        this.successCount = source["successCount"];
	        this.failureCount = source["failureCount"];
	        this.quarantinedCount = source["quarantinedCount"];
	        this.createdAt = source["createdAt"];
	        this.completedAt = source["completedAt"];
	    }
	}
	export class ImportRunDetailDTO {
	    run: ImportRunSummaryDTO;
	    records: ImportRawRecordDetailDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ImportRunDetailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run = this.convertValues(source["run"], ImportRunSummaryDTO);
	        this.records = this.convertValues(source["records"], ImportRawRecordDetailDTO);
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
	export class ImportRunPageDTO {
	    items: ImportRunSummaryDTO[];
	    nextCursor: string;
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportRunPageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ImportRunSummaryDTO);
	        this.nextCursor = source["nextCursor"];
	        this.hasMore = source["hasMore"];
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
	
	export class ImportShipmentEntry {
	    supplierOrderLineId: number;
	    fulfillmentLineId: number;
	    externalShipmentNo: string;
	    carrierCode: string;
	    carrierName: string;
	    trackingNo: string;
	    quantity: number;
	    shippedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportShipmentEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplierOrderLineId = source["supplierOrderLineId"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.externalShipmentNo = source["externalShipmentNo"];
	        this.carrierCode = source["carrierCode"];
	        this.carrierName = source["carrierName"];
	        this.trackingNo = source["trackingNo"];
	        this.quantity = source["quantity"];
	        this.shippedAt = source["shippedAt"];
	    }
	}
	export class ImportShipmentError {
	    entryIndex: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportShipmentError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entryIndex = source["entryIndex"];
	        this.reason = source["reason"];
	    }
	}
	export class ImportShipmentInput {
	    waveId: number;
	    integrationProfileId: number;
	    importMode: string;
	    entries: ImportShipmentEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ImportShipmentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.importMode = source["importMode"];
	        this.entries = this.convertValues(source["entries"], ImportShipmentEntry);
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
	    shippedAt?: string;
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
	export class ImportShipmentResult {
	    importRunId: number;
	    evidenceDisabled: boolean;
	    createdShipments: ShipmentDTO[];
	    errors: ImportShipmentError[];
	    totalProcessed: number;
	    successCount: number;
	    errorCount: number;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportShipmentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importRunId = source["importRunId"];
	        this.evidenceDisabled = source["evidenceDisabled"];
	        this.createdShipments = this.convertValues(source["createdShipments"], ShipmentDTO);
	        this.errors = this.convertValues(source["errors"], ImportShipmentError);
	        this.totalProcessed = source["totalProcessed"];
	        this.successCount = source["successCount"];
	        this.errorCount = source["errorCount"];
	        this.warnings = source["warnings"];
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
	    supportsExportSupplierOrder: boolean;
	    supportsImportProductCatalog: boolean;
	    supportsImportSupplierShipment: boolean;
	    connectorKey: string;
	    factorySupplierPlatform: string;
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
	        this.supportsExportSupplierOrder = source["supportsExportSupplierOrder"];
	        this.supportsImportProductCatalog = source["supportsImportProductCatalog"];
	        this.supportsImportSupplierShipment = source["supportsImportSupplierShipment"];
	        this.connectorKey = source["connectorKey"];
	        this.factorySupplierPlatform = source["factorySupplierPlatform"];
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
	    sourceSurface: string;
	    trackingSyncMode: string;
	    closurePolicy: string;
	    allowsManualClosure: boolean;
	    supportsExportSupplierOrder: boolean;
	    supportsImportProductCatalog: boolean;
	    supportsImportSupplierShipment: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IntegrationProfileSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.profileKey = source["profileKey"];
	        this.sourceChannel = source["sourceChannel"];
	        this.sourceSurface = source["sourceSurface"];
	        this.trackingSyncMode = source["trackingSyncMode"];
	        this.closurePolicy = source["closurePolicy"];
	        this.allowsManualClosure = source["allowsManualClosure"];
	        this.supportsExportSupplierOrder = source["supportsExportSupplierOrder"];
	        this.supportsImportProductCatalog = source["supportsImportProductCatalog"];
	        this.supportsImportSupplierShipment = source["supportsImportSupplierShipment"];
	    }
	}
	export class ListImportRunsPageInput {
	    limit: number;
	    cursor: string;
	    status: string;
	    profileId?: number;
	    documentType: string;
	
	    static createFrom(source: any = {}) {
	        return new ListImportRunsPageInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.limit = source["limit"];
	        this.cursor = source["cursor"];
	        this.status = source["status"];
	        this.profileId = source["profileId"];
	        this.documentType = source["documentType"];
	    }
	}
	export class MapAndReconcileShipmentsInput {
	    waveId: number;
	    integrationProfileId: number;
	    importMode: string;
	    filePath: string;
	    rows: any[];
	
	    static createFrom(source: any = {}) {
	        return new MapAndReconcileShipmentsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.integrationProfileId = source["integrationProfileId"];
	        this.importMode = source["importMode"];
	        this.filePath = source["filePath"];
	        this.rows = source["rows"];
	    }
	}
	export class MarkSupplierOrderSubmittedInput {
	    orderId: number;
	    externalOrderNo: string;
	    submittedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new MarkSupplierOrderSubmittedInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.orderId = source["orderId"];
	        this.externalOrderNo = source["externalOrderNo"];
	        this.submittedAt = source["submittedAt"];
	    }
	}
	
	export class MergeEvidenceDTO {
	    id: number;
	    evidenceKind: string;
	    polarity: string;
	    explanationCode: string;
	    confidence: number;
	    valueHash: string;
	    maskedValue: string;
	    observedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new MergeEvidenceDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.evidenceKind = source["evidenceKind"];
	        this.polarity = source["polarity"];
	        this.explanationCode = source["explanationCode"];
	        this.confidence = source["confidence"];
	        this.valueHash = source["valueHash"];
	        this.maskedValue = source["maskedValue"];
	        this.observedAt = source["observedAt"];
	    }
	}
	export class MergeCandidateDTO {
	    id: number;
	    sourceProfileId: number;
	    targetProfileId: number;
	    status: string;
	    confidence: number;
	    explanationCode: string;
	    evidenceHash: string;
	    policyVersion: number;
	    policyRevisionId?: number;
	    blockerCodes: string[];
	    lastEvaluatedAt?: string;
	    expiresAt?: string;
	    evidence: MergeEvidenceDTO[];
	    createdAt: string;
	    updatedAt: string;
	    rowVersion: number;
	
	    static createFrom(source: any = {}) {
	        return new MergeCandidateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.status = source["status"];
	        this.confidence = source["confidence"];
	        this.explanationCode = source["explanationCode"];
	        this.evidenceHash = source["evidenceHash"];
	        this.policyVersion = source["policyVersion"];
	        this.policyRevisionId = source["policyRevisionId"];
	        this.blockerCodes = source["blockerCodes"];
	        this.lastEvaluatedAt = source["lastEvaluatedAt"];
	        this.expiresAt = source["expiresAt"];
	        this.evidence = this.convertValues(source["evidence"], MergeEvidenceDTO);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.rowVersion = source["rowVersion"];
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
	
	
	
	
	export class MergePolicyRulesDTO {
	    candidateDetectionEnabled: boolean;
	    emailEvidenceMode: string;
	    phoneEvidenceMode: string;
	    executionMode: string;
	
	    static createFrom(source: any = {}) {
	        return new MergePolicyRulesDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidateDetectionEnabled = source["candidateDetectionEnabled"];
	        this.emailEvidenceMode = source["emailEvidenceMode"];
	        this.phoneEvidenceMode = source["phoneEvidenceMode"];
	        this.executionMode = source["executionMode"];
	    }
	}
	export class MergePolicyDTO {
	    id: number;
	    policyKey: string;
	    revision: number;
	    rules: MergePolicyRulesDTO;
	    needsScan: boolean;
	    lastScanAt?: string;
	    revisionTime: string;
	
	    static createFrom(source: any = {}) {
	        return new MergePolicyDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.policyKey = source["policyKey"];
	        this.revision = source["revision"];
	        this.rules = this.convertValues(source["rules"], MergePolicyRulesDTO);
	        this.needsScan = source["needsScan"];
	        this.lastScanAt = source["lastScanAt"];
	        this.revisionTime = source["revisionTime"];
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
	
	export class MergePreviewAddress {
	    id: number;
	    label: string;
	    recipientName: string;
	    phone: string;
	    country: string;
	    province: string;
	    city: string;
	    district: string;
	    addressLine1: string;
	    addressLine2: string;
	    postalCode: string;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergePreviewAddress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.recipientName = source["recipientName"];
	        this.phone = source["phone"];
	        this.country = source["country"];
	        this.province = source["province"];
	        this.city = source["city"];
	        this.district = source["district"];
	        this.addressLine1 = source["addressLine1"];
	        this.addressLine2 = source["addressLine2"];
	        this.postalCode = source["postalCode"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class MergePreviewConflict {
	    field: string;
	    sourceValue: string;
	    targetValue: string;
	
	    static createFrom(source: any = {}) {
	        return new MergePreviewConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.sourceValue = source["sourceValue"];
	        this.targetValue = source["targetValue"];
	    }
	}
	export class MergePreviewIdentity {
	    id: number;
	    identityPlatform: string;
	    identityValue: string;
	    identityType: string;
	    isPrimary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergePreviewIdentity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.identityPlatform = source["identityPlatform"];
	        this.identityValue = source["identityValue"];
	        this.identityType = source["identityType"];
	        this.isPrimary = source["isPrimary"];
	    }
	}
	export class MergePreviewProfileSide {
	    profileId: number;
	    displayName: string;
	    profileType: string;
	    identities: MergePreviewIdentity[];
	    addresses: MergePreviewAddress[];
	
	    static createFrom(source: any = {}) {
	        return new MergePreviewProfileSide(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.displayName = source["displayName"];
	        this.profileType = source["profileType"];
	        this.identities = this.convertValues(source["identities"], MergePreviewIdentity);
	        this.addresses = this.convertValues(source["addresses"], MergePreviewAddress);
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
	export class MergeProfilesInput {
	    sourceProfileId: number;
	    targetProfileId: number;
	
	    static createFrom(source: any = {}) {
	        return new MergeProfilesInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	    }
	}
	export class MergeProfilesPreviewResult {
	    source: MergePreviewProfileSide;
	    target: MergePreviewProfileSide;
	    conflicts: MergePreviewConflict[];
	    movedIdentityCount: number;
	    movedAddressCount: number;
	    duplicateIdentityValues: string[];
	
	    static createFrom(source: any = {}) {
	        return new MergeProfilesPreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = this.convertValues(source["source"], MergePreviewProfileSide);
	        this.target = this.convertValues(source["target"], MergePreviewProfileSide);
	        this.conflicts = this.convertValues(source["conflicts"], MergePreviewConflict);
	        this.movedIdentityCount = source["movedIdentityCount"];
	        this.movedAddressCount = source["movedAddressCount"];
	        this.duplicateIdentityValues = source["duplicateIdentityValues"];
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
	export class MergeProfilesResult {
	    migratedIdentityCount: number;
	    migratedAddressCount: number;
	    updatedDemandDocs: number;
	    updatedParticipants: number;
	    updatedFulfillmentLines: number;
	    mergeId: number;
	    undoAvailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergeProfilesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.migratedIdentityCount = source["migratedIdentityCount"];
	        this.migratedAddressCount = source["migratedAddressCount"];
	        this.updatedDemandDocs = source["updatedDemandDocs"];
	        this.updatedParticipants = source["updatedParticipants"];
	        this.updatedFulfillmentLines = source["updatedFulfillmentLines"];
	        this.mergeId = source["mergeId"];
	        this.undoAvailable = source["undoAvailable"];
	    }
	}
	export class MergeScanRunDTO {
	    id: number;
	    policyVersion: number;
	    status: string;
	    startedAt: string;
	    completedAt?: string;
	    profilesScanned: number;
	    pairsEvaluated: number;
	    candidatesCreated: number;
	    candidatesUpdated: number;
	    candidatesBlocked: number;
	    errorMessage: string;
	
	    static createFrom(source: any = {}) {
	        return new MergeScanRunDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.policyVersion = source["policyVersion"];
	        this.status = source["status"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	        this.profilesScanned = source["profilesScanned"];
	        this.pairsEvaluated = source["pairsEvaluated"];
	        this.candidatesCreated = source["candidatesCreated"];
	        this.candidatesUpdated = source["candidatesUpdated"];
	        this.candidatesBlocked = source["candidatesBlocked"];
	        this.errorMessage = source["errorMessage"];
	    }
	}
	export class MergeSuggestionDTO {
	    id: number;
	    sourceProfileId: number;
	    targetProfileId: number;
	    reason: string;
	    status: string;
	    sourceProfile: CustomerProfileDTO;
	    targetProfile: CustomerProfileDTO;
	
	    static createFrom(source: any = {}) {
	        return new MergeSuggestionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceProfileId = source["sourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.reason = source["reason"];
	        this.status = source["status"];
	        this.sourceProfile = this.convertValues(source["sourceProfile"], CustomerProfileDTO);
	        this.targetProfile = this.convertValues(source["targetProfile"], CustomerProfileDTO);
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
	export class PaginationInput {
	    page: number;
	    pageSize: number;
	    sortBy: string;
	    sortDesc: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PaginationInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.sortBy = source["sortBy"];
	        this.sortDesc = source["sortDesc"];
	    }
	}
	
	export class PinCustomerDisplayNameInput {
	    profileId: number;
	    name: string;
	    expectedRowVersion: number;
	    actorRef: string;
	    idempotencyKey: string;
	
	    static createFrom(source: any = {}) {
	        return new PinCustomerDisplayNameInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.name = source["name"];
	        this.expectedRowVersion = source["expectedRowVersion"];
	        this.actorRef = source["actorRef"];
	        this.idempotencyKey = source["idempotencyKey"];
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
	
	export class ProductMasterPageFilterInput {
	    keyword: string;
	    productKinds: string[];
	    archivedOnly: boolean;
	    sortBy: string;
	    sortDir: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductMasterPageFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.productKinds = source["productKinds"];
	        this.archivedOnly = source["archivedOnly"];
	        this.sortBy = source["sortBy"];
	        this.sortDir = source["sortDir"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class ProductMasterPageResult {
	    items: ProductMasterDTO[];
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductMasterPageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ProductMasterDTO);
	        this.totalCount = source["totalCount"];
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
	    adjustmentId: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ReplayFailureDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.adjustmentId = source["adjustmentId"];
	        this.reason = source["reason"];
	    }
	}
	export class ReconcileResultDTO {
	    created: number;
	    deleted: number;
	    replayedCount: number;
	    failures: ReplayFailureDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ReconcileResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = source["created"];
	        this.deleted = source["deleted"];
	        this.replayedCount = source["replayedCount"];
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
	export class SupplierOrderLineAcceptanceEntry {
	    lineId: number;
	    acceptedQuantity: number;
	
	    static createFrom(source: any = {}) {
	        return new SupplierOrderLineAcceptanceEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lineId = source["lineId"];
	        this.acceptedQuantity = source["acceptedQuantity"];
	    }
	}
	export class RecordSupplierOrderAcceptanceInput {
	    orderId: number;
	    lines: SupplierOrderLineAcceptanceEntry[];
	
	    static createFrom(source: any = {}) {
	        return new RecordSupplierOrderAcceptanceInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.orderId = source["orderId"];
	        this.lines = this.convertValues(source["lines"], SupplierOrderLineAcceptanceEntry);
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
	export class RegisterExternalCarrierInput {
	    integrationProfileId: number;
	    externalCarrierCode: string;
	    externalCarrierName: string;
	
	    static createFrom(source: any = {}) {
	        return new RegisterExternalCarrierInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.integrationProfileId = source["integrationProfileId"];
	        this.externalCarrierCode = source["externalCarrierCode"];
	        this.externalCarrierName = source["externalCarrierName"];
	    }
	}
	
	export class SetImportEvidenceRetentionInput {
	    retentionDays: number;
	
	    static createFrom(source: any = {}) {
	        return new SetImportEvidenceRetentionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.retentionDays = source["retentionDays"];
	    }
	}
	export class ShipmentByWavePageFilterInput {
	    waveId: number;
	    sortBy: string;
	    sortDir: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new ShipmentByWavePageFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.sortBy = source["sortBy"];
	        this.sortDir = source["sortDir"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	
	
	export class ShipmentPageResult {
	    items: ShipmentDTO[];
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ShipmentPageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ShipmentDTO);
	        this.totalCount = source["totalCount"];
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
	export class SnapshotProductDetailItem {
	    masterId: number;
	    product: ProductDTO;
	    alreadyExisted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotProductDetailItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.masterId = source["masterId"];
	        this.product = this.convertValues(source["product"], ProductDTO);
	        this.alreadyExisted = source["alreadyExisted"];
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
	export class SnapshotProductsDetailedResult {
	    items: SnapshotProductDetailItem[];
	    createdCount: number;
	    skippedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotProductsDetailedResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], SnapshotProductDetailItem);
	        this.createdCount = source["createdCount"];
	        this.skippedCount = source["skippedCount"];
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
	    factoryIntegrationProfileId?: number;
	    supplierPlatform: string;
	    templateId: string;
	    batchNo: string;
	    externalOrderNo: string;
	    submissionMode: string;
	    submittedAt?: string;
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
	        this.factoryIntegrationProfileId = source["factoryIntegrationProfileId"];
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
	export class SupplierOrderFileResultDTO {
	    orderId: number;
	    filePath: string;
	    lineCount: number;
	    generatedAt: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SupplierOrderFileResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.orderId = source["orderId"];
	        this.filePath = source["filePath"];
	        this.lineCount = source["lineCount"];
	        this.generatedAt = source["generatedAt"];
	        this.warnings = source["warnings"];
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
	export class SupplierOrderLineShippedDTO {
	    lineId: number;
	    fulfillmentLineId: number;
	    batchNo: string;
	    supplierLineNo: number;
	    supplierSku: string;
	    submittedQuantity: number;
	    acceptedQuantity: number;
	    shippedQuantity: number;
	    remainingQuantity: number;
	
	    static createFrom(source: any = {}) {
	        return new SupplierOrderLineShippedDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lineId = source["lineId"];
	        this.fulfillmentLineId = source["fulfillmentLineId"];
	        this.batchNo = source["batchNo"];
	        this.supplierLineNo = source["supplierLineNo"];
	        this.supplierSku = source["supplierSku"];
	        this.submittedQuantity = source["submittedQuantity"];
	        this.acceptedQuantity = source["acceptedQuantity"];
	        this.shippedQuantity = source["shippedQuantity"];
	        this.remainingQuantity = source["remainingQuantity"];
	    }
	}
	export class SystemSettingsDTO {
	    autoMergeCrossPlatform: boolean;
	    autoMergeByEmail: boolean;
	    autoMergeByPhone: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SystemSettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoMergeCrossPlatform = source["autoMergeCrossPlatform"];
	        this.autoMergeByEmail = source["autoMergeByEmail"];
	        this.autoMergeByPhone = source["autoMergeByPhone"];
	    }
	}
	export class UnassignDemandInput {
	    waveId: number;
	    demandDocumentId: number;
	
	    static createFrom(source: any = {}) {
	        return new UnassignDemandInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.demandDocumentId = source["demandDocumentId"];
	    }
	}
	export class UndoCustomerMergeInput {
	    mergeId: number;
	
	    static createFrom(source: any = {}) {
	        return new UndoCustomerMergeInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	    }
	}
	export class UndoCustomerMergeResult {
	    mergeId: number;
	    restoredSourceProfileId: number;
	    targetProfileId: number;
	    restoredIdentityCount: number;
	    restoredAddressCount: number;
	    restoredDemandDocumentCount: number;
	
	    static createFrom(source: any = {}) {
	        return new UndoCustomerMergeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mergeId = source["mergeId"];
	        this.restoredSourceProfileId = source["restoredSourceProfileId"];
	        this.targetProfileId = source["targetProfileId"];
	        this.restoredIdentityCount = source["restoredIdentityCount"];
	        this.restoredAddressCount = source["restoredAddressCount"];
	        this.restoredDemandDocumentCount = source["restoredDemandDocumentCount"];
	    }
	}
	export class UnpinCustomerDisplayNameInput {
	    profileId: number;
	    expectedRowVersion: number;
	    actorRef: string;
	    idempotencyKey: string;
	
	    static createFrom(source: any = {}) {
	        return new UnpinCustomerDisplayNameInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.expectedRowVersion = source["expectedRowVersion"];
	        this.actorRef = source["actorRef"];
	        this.idempotencyKey = source["idempotencyKey"];
	    }
	}
	export class UpdateAddressInput {
	    id: number;
	    customerProfileId: number;
	    label: string;
	    recipientName: string;
	    phone: string;
	    country: string;
	    province: string;
	    city: string;
	    district: string;
	    addressLine1: string;
	    addressLine2: string;
	    postalCode: string;
	    isDefault: boolean;
	    isTest: boolean;
	    validationStatus: string;
	    validationDetail: string;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateAddressInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customerProfileId = source["customerProfileId"];
	        this.label = source["label"];
	        this.recipientName = source["recipientName"];
	        this.phone = source["phone"];
	        this.country = source["country"];
	        this.province = source["province"];
	        this.city = source["city"];
	        this.district = source["district"];
	        this.addressLine1 = source["addressLine1"];
	        this.addressLine2 = source["addressLine2"];
	        this.postalCode = source["postalCode"];
	        this.isDefault = source["isDefault"];
	        this.isTest = source["isTest"];
	        this.validationStatus = source["validationStatus"];
	        this.validationDetail = source["validationDetail"];
	        this.extraData = source["extraData"];
	    }
	}
	export class UpdateAllocationPolicyRuleInput {
	    id: number;
	    productId?: number;
	    selectorPayload?: number[];
	    productTargetRef?: string;
	    contributionQuantity?: number;
	    ruleKind?: string;
	    priority?: number;
	    active?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateAllocationPolicyRuleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.productId = source["productId"];
	        this.selectorPayload = source["selectorPayload"];
	        this.productTargetRef = source["productTargetRef"];
	        this.contributionQuantity = source["contributionQuantity"];
	        this.ruleKind = source["ruleKind"];
	        this.priority = source["priority"];
	        this.active = source["active"];
	    }
	}
	export class UpdateCustomerProfileInput {
	    id: number;
	    displayName: string;
	    profileType: string;
	    extraData: string;
	    expectedRowVersion: number;
	    actorRef: string;
	    idempotencyKey: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCustomerProfileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.profileType = source["profileType"];
	        this.extraData = source["extraData"];
	        this.expectedRowVersion = source["expectedRowVersion"];
	        this.actorRef = source["actorRef"];
	        this.idempotencyKey = source["idempotencyKey"];
	    }
	}
	export class UpdateCustomerResolutionFeaturePolicyInput {
	    expectedRevision: number;
	    customerResolutionWritesEnabled: boolean;
	    candidateScanEnabled: boolean;
	    mergeExecutionEnabled: boolean;
	    splitExecutionEnabled: boolean;
	    importEvidenceEnabled: boolean;
	    carrierRegistryWritesEnabled: boolean;
	    actorRef: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCustomerResolutionFeaturePolicyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.customerResolutionWritesEnabled = source["customerResolutionWritesEnabled"];
	        this.candidateScanEnabled = source["candidateScanEnabled"];
	        this.mergeExecutionEnabled = source["mergeExecutionEnabled"];
	        this.splitExecutionEnabled = source["splitExecutionEnabled"];
	        this.importEvidenceEnabled = source["importEvidenceEnabled"];
	        this.carrierRegistryWritesEnabled = source["carrierRegistryWritesEnabled"];
	        this.actorRef = source["actorRef"];
	        this.reason = source["reason"];
	    }
	}
	
	export class UpdateDocumentTemplateInput {
	    id: number;
	    format: string;
	    mappingRules: string;
	    extraData: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateDocumentTemplateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.format = source["format"];
	        this.mappingRules = source["mappingRules"];
	        this.extraData = source["extraData"];
	    }
	}
	export class UpdateMergePolicyInput {
	    expectedRevision: number;
	    rules: MergePolicyRulesDTO;
	    actorRef: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateMergePolicyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.rules = this.convertValues(source["rules"], MergePolicyRulesDTO);
	        this.actorRef = source["actorRef"];
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
	    coverImagePath: string;
	    detailImagePaths: string;
	    extraData: string;
	
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
	        this.coverImagePath = source["coverImagePath"];
	        this.detailImagePaths = source["detailImagePaths"];
	        this.extraData = source["extraData"];
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
	    supportsExportSupplierOrder: boolean;
	    supportsImportProductCatalog: boolean;
	    supportsImportSupplierShipment: boolean;
	    connectorKey: string;
	    factorySupplierPlatform: string;
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
	        this.supportsExportSupplierOrder = source["supportsExportSupplierOrder"];
	        this.supportsImportProductCatalog = source["supportsImportProductCatalog"];
	        this.supportsImportSupplierShipment = source["supportsImportSupplierShipment"];
	        this.connectorKey = source["connectorKey"];
	        this.factorySupplierPlatform = source["factorySupplierPlatform"];
	        this.supportedLocales = source["supportedLocales"];
	        this.defaultLocale = source["defaultLocale"];
	        this.extraData = source["extraData"];
	    }
	}
	export class UpdateShipmentInput {
	    id: number;
	    supplierPlatform: string;
	    shipmentNo: string;
	    externalShipmentNo: string;
	    carrierCode: string;
	    carrierName: string;
	    trackingNo: string;
	    shippedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateShipmentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.supplierPlatform = source["supplierPlatform"];
	        this.shipmentNo = source["shipmentNo"];
	        this.externalShipmentNo = source["externalShipmentNo"];
	        this.carrierCode = source["carrierCode"];
	        this.carrierName = source["carrierName"];
	        this.trackingNo = source["trackingNo"];
	        this.shippedAt = source["shippedAt"];
	    }
	}
	export class UpdateWaveInput {
	    waveId: number;
	    name: string;
	    notes: string;
	    levelTags: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWaveInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.name = source["name"];
	        this.notes = source["notes"];
	        this.levelTags = source["levelTags"];
	    }
	}
	export class VoidShipmentInput {
	    id: number;
	    note: string;
	    operatorId: string;
	
	    static createFrom(source: any = {}) {
	        return new VoidShipmentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.note = source["note"];
	        this.operatorId = source["operatorId"];
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
	export class WaveFulfillmentFilterInput {
	    waveId: number;
	    allocationStates: string[];
	    addressStates: string[];
	    supplierStates: string[];
	    channelSyncStates: string[];
	    reviewRequirements: string[];
	    driftStatuses: string[];
	    keyword: string;
	    pagination: PaginationInput;
	
	    static createFrom(source: any = {}) {
	        return new WaveFulfillmentFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.waveId = source["waveId"];
	        this.allocationStates = source["allocationStates"];
	        this.addressStates = source["addressStates"];
	        this.supplierStates = source["supplierStates"];
	        this.channelSyncStates = source["channelSyncStates"];
	        this.reviewRequirements = source["reviewRequirements"];
	        this.driftStatuses = source["driftStatuses"];
	        this.keyword = source["keyword"];
	        this.pagination = this.convertValues(source["pagination"], PaginationInput);
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
	export class WaveFulfillmentRowsPage {
	    items: WaveFulfillmentRowDTO[];
	    pagination: PaginationResult;
	
	    static createFrom(source: any = {}) {
	        return new WaveFulfillmentRowsPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], WaveFulfillmentRowDTO);
	        this.pagination = this.convertValues(source["pagination"], PaginationResult);
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
	export class WaveOverviewDTO {
	    wave: WaveDTO;
	    demandKinds: string[];
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
	    fulfillmentDraftCount: number;
	    fulfillmentReadyCount: number;
	    addressMissingCount: number;
	    addressReadyCount: number;
	    addressInvalidCount: number;
	    supplierNotSubmittedCount: number;
	    supplierSubmittedCount: number;
	    supplierShippedCount: number;
	    adjustmentCount: number;
	    adjustmentAddCount: number;
	    adjustmentReduceCount: number;
	    adjustmentReplaceCount: number;
	    adjustmentRemoveCount: number;
	    replayHealthy: boolean;
	    replayFailureCount: number;
	    suggestedNextStep: string;
	    nextStepReason: string;
	    blockingIssues: string[];
	
	    static createFrom(source: any = {}) {
	        return new WaveOverviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wave = this.convertValues(source["wave"], WaveDTO);
	        this.demandKinds = source["demandKinds"];
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
	        this.fulfillmentDraftCount = source["fulfillmentDraftCount"];
	        this.fulfillmentReadyCount = source["fulfillmentReadyCount"];
	        this.addressMissingCount = source["addressMissingCount"];
	        this.addressReadyCount = source["addressReadyCount"];
	        this.addressInvalidCount = source["addressInvalidCount"];
	        this.supplierNotSubmittedCount = source["supplierNotSubmittedCount"];
	        this.supplierSubmittedCount = source["supplierSubmittedCount"];
	        this.supplierShippedCount = source["supplierShippedCount"];
	        this.adjustmentCount = source["adjustmentCount"];
	        this.adjustmentAddCount = source["adjustmentAddCount"];
	        this.adjustmentReduceCount = source["adjustmentReduceCount"];
	        this.adjustmentReplaceCount = source["adjustmentReplaceCount"];
	        this.adjustmentRemoveCount = source["adjustmentRemoveCount"];
	        this.replayHealthy = source["replayHealthy"];
	        this.replayFailureCount = source["replayFailureCount"];
	        this.suggestedNextStep = source["suggestedNextStep"];
	        this.nextStepReason = source["nextStepReason"];
	        this.blockingIssues = source["blockingIssues"];
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
	    demandKinds: string[];
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
	        this.demandKinds = source["demandKinds"];
	        this.fulfillmentLineCount = source["fulfillmentLineCount"];
	        this.readyFulfillmentCount = source["readyFulfillmentCount"];
	    }
	}
	export class WaveRoutingStatsDTO {
	    totalLines: number;
	    acceptedReadyCount: number;
	    acceptedWaitingCount: number;
	    acceptedPartialCount: number;
	    deferredCount: number;
	    excludedManualCount: number;
	    excludedDuplicateCount: number;
	    excludedRevokedCount: number;
	    pendingIntakeCount: number;
	
	    static createFrom(source: any = {}) {
	        return new WaveRoutingStatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalLines = source["totalLines"];
	        this.acceptedReadyCount = source["acceptedReadyCount"];
	        this.acceptedWaitingCount = source["acceptedWaitingCount"];
	        this.acceptedPartialCount = source["acceptedPartialCount"];
	        this.deferredCount = source["deferredCount"];
	        this.excludedManualCount = source["excludedManualCount"];
	        this.excludedDuplicateCount = source["excludedDuplicateCount"];
	        this.excludedRevokedCount = source["excludedRevokedCount"];
	        this.pendingIntakeCount = source["pendingIntakeCount"];
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
	    historyHeadNodeId: number;
	    historyHeadProjectionHash: string;
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
	        this.historyHeadNodeId = source["historyHeadNodeId"];
	        this.historyHeadProjectionHash = source["historyHeadProjectionHash"];
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
	export class WavesPage {
	    items: WaveDTO[];
	    pagination: PaginationResult;
	
	    static createFrom(source: any = {}) {
	        return new WavesPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], WaveDTO);
	        this.pagination = this.convertValues(source["pagination"], PaginationResult);
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

