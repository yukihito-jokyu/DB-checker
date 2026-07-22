export namespace wails {
	
	export class ProfileResponse {
	    id: string;
	    name: string;
	    dbType: string;
	    host: string;
	    port: number;
	    database: string;
	    schema: string;
	    user: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.dbType = source["dbType"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.schema = source["schema"];
	        this.user = source["user"];
	    }
	}
	export class ConfigResponse {
	    version: number;
	    connectionProfiles: ProfileResponse[];
	    activeConnectionProfileId?: string;
	    flowStates: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ConfigResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.connectionProfiles = this.convertValues(source["connectionProfiles"], ProfileResponse);
	        this.activeConnectionProfileId = source["activeConnectionProfileId"];
	        this.flowStates = source["flowStates"];
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
	export class ProfileCheckResponse {
	    valid: boolean;
	    profileCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileCheckResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.profileCount = source["profileCount"];
	    }
	}
	
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_ConfigResponse_ {
	    data?: ConfigResponse;
	    error?: ErrorResponse;
	
	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_ConfigResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], ConfigResponse);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_ProfileCheckResponse_ {
	    data?: ProfileCheckResponse;
	    error?: ErrorResponse;
	
	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_ProfileCheckResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], ProfileCheckResponse);
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
	export class StatusResponse {
	    name: string;
	    ready: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ready = source["ready"];
	        this.version = source["version"];
	    }
	}
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_StatusResponse_ {
	    data?: StatusResponse;
	    error?: ErrorResponse;
	
	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_StatusResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], StatusResponse);
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

