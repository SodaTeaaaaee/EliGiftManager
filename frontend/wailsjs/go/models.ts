export namespace main {
	
	export class BatchSummary {
	    batchName: string;
	    totalRecords: number;
	    totalQuantity: number;
	    pendingAddressRecords: number;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BatchSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.batchName = source["batchName"];
	        this.totalRecords = source["totalRecords"];
	        this.totalQuantity = source["totalQuantity"];
	        this.pendingAddressRecords = source["pendingAddressRecords"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class BootstrapPayload {
	    name: string;
	    version: string;
	    module: string;
	    description: string;
	    runtime: string;
	    frontend: string;
	    highlights: string[];
	
	    static createFrom(source: any = {}) {
	        return new BootstrapPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.module = source["module"];
	        this.description = source["description"];
	        this.runtime = source["runtime"];
	        this.frontend = source["frontend"];
	        this.highlights = source["highlights"];
	    }
	}
	export class DashboardWarning {
	    title: string;
	    detail: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardWarning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.type = source["type"];
	    }
	}
	export class DispatchRecordItem {
	    id: number;
	    batchName: string;
	    quantity: number;
	    status: string;
	    memberId: number;
	    platform: string;
	    platformUid: string;
	    memberNickname: string;
	    productId: number;
	    productName: string;
	    factorySku: string;
	    recipientName: string;
	    phone: string;
	    address: string;
	    hasAddress: boolean;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new DispatchRecordItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.batchName = source["batchName"];
	        this.quantity = source["quantity"];
	        this.status = source["status"];
	        this.memberId = source["memberId"];
	        this.platform = source["platform"];
	        this.platformUid = source["platformUid"];
	        this.memberNickname = source["memberNickname"];
	        this.productId = source["productId"];
	        this.productName = source["productName"];
	        this.factorySku = source["factorySku"];
	        this.recipientName = source["recipientName"];
	        this.phone = source["phone"];
	        this.address = source["address"];
	        this.hasAddress = source["hasAddress"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class DashboardPayload {
	    databasePath: string;
	    memberCount: number;
	    productCount: number;
	    dispatchCount: number;
	    templateCount: number;
	    addressCount: number;
	    missingAddresses: number;
	    pendingAddresses: number;
	    batchCount: number;
	    batches: BatchSummary[];
	    recentDispatches: DispatchRecordItem[];
	    warnings: DashboardWarning[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.databasePath = source["databasePath"];
	        this.memberCount = source["memberCount"];
	        this.productCount = source["productCount"];
	        this.dispatchCount = source["dispatchCount"];
	        this.templateCount = source["templateCount"];
	        this.addressCount = source["addressCount"];
	        this.missingAddresses = source["missingAddresses"];
	        this.pendingAddresses = source["pendingAddresses"];
	        this.batchCount = source["batchCount"];
	        this.batches = this.convertValues(source["batches"], BatchSummary);
	        this.recentDispatches = this.convertValues(source["recentDispatches"], DispatchRecordItem);
	        this.warnings = this.convertValues(source["warnings"], DashboardWarning);
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
	
	
	export class MemberItem {
	    id: number;
	    platform: string;
	    platformUid: string;
	    latestNickname: string;
	    addressCount: number;
	    activeAddressCount: number;
	    latestRecipient: string;
	    latestPhone: string;
	    latestAddress: string;
	    dispatchCount: number;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new MemberItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.platformUid = source["platformUid"];
	        this.latestNickname = source["latestNickname"];
	        this.addressCount = source["addressCount"];
	        this.activeAddressCount = source["activeAddressCount"];
	        this.latestRecipient = source["latestRecipient"];
	        this.latestPhone = source["latestPhone"];
	        this.latestAddress = source["latestAddress"];
	        this.dispatchCount = source["dispatchCount"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	    id: number;
	    factory: string;
	    factorySku: string;
	    name: string;
	    imagePath: string;
	    extraData: string;
	    dispatchCount: number;
	    totalQuantity: number;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ProductItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.factory = source["factory"];
	        this.factorySku = source["factorySku"];
	        this.name = source["name"];
	        this.imagePath = source["imagePath"];
	        this.extraData = source["extraData"];
	        this.dispatchCount = source["dispatchCount"];
	        this.totalQuantity = source["totalQuantity"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class TemplateItem {
	    id: number;
	    type: string;
	    name: string;
	    mappingRules: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new TemplateItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.mappingRules = source["mappingRules"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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

export namespace model {
	
	export class BatchValidationMissingMember {
	    memberId: number;
	    platform: string;
	    platformUid: string;
	    latestNickname: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchValidationMissingMember(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.memberId = source["memberId"];
	        this.platform = source["platform"];
	        this.platformUid = source["platformUid"];
	        this.latestNickname = source["latestNickname"];
	    }
	}
	export class BatchValidationResult {
	    batchName: string;
	    totalRecords: number;
	    boundAddressRecords: number;
	    pendingAddressRecords: number;
	    missingMembers: BatchValidationMissingMember[];
	
	    static createFrom(source: any = {}) {
	        return new BatchValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.batchName = source["batchName"];
	        this.totalRecords = source["totalRecords"];
	        this.boundAddressRecords = source["boundAddressRecords"];
	        this.pendingAddressRecords = source["pendingAddressRecords"];
	        this.missingMembers = this.convertValues(source["missingMembers"], BatchValidationMissingMember);
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

