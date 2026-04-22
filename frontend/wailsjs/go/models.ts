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

