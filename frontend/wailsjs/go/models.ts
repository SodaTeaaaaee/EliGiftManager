export namespace main {
	
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
	    waveId: number;
	    waveNo: string;
	    quantity: number;
	    status: string;
	    memberId: number;
	    platform: string;
	    platformUid: string;
	    memberNickname: string;
	    productId: number;
	    productName: string;
	    factorySku: string;
	    memberAddressId?: number;
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
	        this.waveId = source["waveId"];
	        this.waveNo = source["waveNo"];
	        this.quantity = source["quantity"];
	        this.status = source["status"];
	        this.memberId = source["memberId"];
	        this.platform = source["platform"];
	        this.platformUid = source["platformUid"];
	        this.memberNickname = source["memberNickname"];
	        this.productId = source["productId"];
	        this.productName = source["productName"];
	        this.factorySku = source["factorySku"];
	        this.memberAddressId = source["memberAddressId"];
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
	export class WaveItem {
	    id: number;
	    waveNo: string;
	    name: string;
	    status: string;
	    totalRecords: number;
	    totalQuantity: number;
	    pendingAddressRecords: number;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new WaveItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveNo = source["waveNo"];
	        this.name = source["name"];
	        this.status = source["status"];
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
	export class DashboardPayload {
	    databasePath: string;
	    memberCount: number;
	    productCount: number;
	    dispatchCount: number;
	    templateCount: number;
	    addressCount: number;
	    missingAddresses: number;
	    pendingAddresses: number;
	    waveCount: number;
	    recentWaves: WaveItem[];
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
	        this.waveCount = source["waveCount"];
	        this.recentWaves = this.convertValues(source["recentWaves"], WaveItem);
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
	    extraData: string;
	    addressCount: number;
	    activeAddressCount: number;
	    latestRecipient: string;
	    latestPhone: string;
	    latestAddress: string;
	    dispatchCount: number;
	    // Go type: time
	    updatedAt: any;
	    addresses: model.MemberAddress[];
	    nicknames: model.MemberNickname[];
	
	    static createFrom(source: any = {}) {
	        return new MemberItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.platformUid = source["platformUid"];
	        this.latestNickname = source["latestNickname"];
	        this.extraData = source["extraData"];
	        this.addressCount = source["addressCount"];
	        this.activeAddressCount = source["activeAddressCount"];
	        this.latestRecipient = source["latestRecipient"];
	        this.latestPhone = source["latestPhone"];
	        this.latestAddress = source["latestAddress"];
	        this.dispatchCount = source["dispatchCount"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.addresses = this.convertValues(source["addresses"], model.MemberAddress);
	        this.nicknames = this.convertValues(source["nicknames"], model.MemberNickname);
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
	export class MemberListPayload {
	    items: MemberItem[];
	    total: number;
	    platforms: string[];
	
	    static createFrom(source: any = {}) {
	        return new MemberListPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], MemberItem);
	        this.total = source["total"];
	        this.platforms = source["platforms"];
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
	    platform: string;
	    factory: string;
	    factorySku: string;
	    name: string;
	    coverImage: string;
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
	        this.platform = source["platform"];
	        this.factory = source["factory"];
	        this.factorySku = source["factorySku"];
	        this.name = source["name"];
	        this.coverImage = source["coverImage"];
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
	export class ProductListPayload {
	    items: ProductItem[];
	    total: number;
	    platforms: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProductListPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ProductItem);
	        this.total = source["total"];
	        this.platforms = source["platforms"];
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
	    platform: string;
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
	        this.platform = source["platform"];
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
	
	export class Product {
	    id: number;
	    platform: string;
	    factory: string;
	    factorySku: string;
	    name: string;
	    coverImage: string;
	    extraData: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Product(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.factory = source["factory"];
	        this.factorySku = source["factorySku"];
	        this.name = source["name"];
	        this.coverImage = source["coverImage"];
	        this.extraData = source["extraData"];
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
	export class MemberAddress {
	    id: number;
	    memberId: number;
	    recipientName: string;
	    phone: string;
	    address: string;
	    isDefault: boolean;
	    isDeleted: boolean;
	    // Go type: time
	    createdAt: any;
	    member: Member;
	
	    static createFrom(source: any = {}) {
	        return new MemberAddress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.memberId = source["memberId"];
	        this.recipientName = source["recipientName"];
	        this.phone = source["phone"];
	        this.address = source["address"];
	        this.isDefault = source["isDefault"];
	        this.isDeleted = source["isDeleted"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.member = this.convertValues(source["member"], Member);
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
	export class MemberNickname {
	    id: number;
	    memberId: number;
	    nickname: string;
	    // Go type: time
	    createdAt: any;
	    member: Member;
	
	    static createFrom(source: any = {}) {
	        return new MemberNickname(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.memberId = source["memberId"];
	        this.nickname = source["nickname"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.member = this.convertValues(source["member"], Member);
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
	export class Member {
	    id: number;
	    platform: string;
	    platformUid: string;
	    extraData: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    nicknames: MemberNickname[];
	    addresses: MemberAddress[];
	
	    static createFrom(source: any = {}) {
	        return new Member(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.platformUid = source["platformUid"];
	        this.extraData = source["extraData"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.nicknames = this.convertValues(source["nicknames"], MemberNickname);
	        this.addresses = this.convertValues(source["addresses"], MemberAddress);
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
	    id: number;
	    waveNo: string;
	    name: string;
	    status: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    records: DispatchRecord[];
	
	    static createFrom(source: any = {}) {
	        return new Wave(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveNo = source["waveNo"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.records = this.convertValues(source["records"], DispatchRecord);
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
	export class DispatchRecord {
	    id: number;
	    waveId: number;
	    memberId: number;
	    productId: number;
	    memberAddressId?: number;
	    quantity: number;
	    status: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    wave: Wave;
	    member: Member;
	    product: Product;
	    memberAddress?: MemberAddress;
	
	    static createFrom(source: any = {}) {
	        return new DispatchRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.waveId = source["waveId"];
	        this.memberId = source["memberId"];
	        this.productId = source["productId"];
	        this.memberAddressId = source["memberAddressId"];
	        this.quantity = source["quantity"];
	        this.status = source["status"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.wave = this.convertValues(source["wave"], Wave);
	        this.member = this.convertValues(source["member"], Member);
	        this.product = this.convertValues(source["product"], Product);
	        this.memberAddress = this.convertValues(source["memberAddress"], MemberAddress);
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

