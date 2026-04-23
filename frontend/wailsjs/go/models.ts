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

