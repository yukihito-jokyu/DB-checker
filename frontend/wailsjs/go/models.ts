export namespace wails {
	
	export class ErrorResponse {
	    code: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ErrorResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class StatusData {
	    name: string;
	    ready: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ready = source["ready"];
	        this.version = source["version"];
	    }
	}
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_StatusData_ {
	    data?: StatusData;
	    error?: ErrorResponse;
	
	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_StatusData_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], StatusData);
	        this.error = this.convertValues(source["error"], ErrorResponse);
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

