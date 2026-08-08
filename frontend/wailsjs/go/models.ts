export namespace wails {

	export class ActiveProfileResponse {
	    id: string;
	    name: string;
	    dbType: string;
	    database: string;
	    schema: string;

	    static createFrom(source: any = {}) {
	        return new ActiveProfileResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.dbType = source["dbType"];
	        this.database = source["database"];
	        this.schema = source["schema"];
	    }
	}
	export class AffectedRowsResponse {
	    affectedRows: number;

	    static createFrom(source: any = {}) {
	        return new AffectedRowsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.affectedRows = source["affectedRows"];
	    }
	}
	export class StatisticValueResponse {
	    value?: string;
	    status: string;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new StatisticValueResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	    }
	}
	export class StatisticCountResponse {
	    value?: number;
	    status: string;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new StatisticCountResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	    }
	}
	export class ColumnStatisticsResponse {
	    name: string;
	    nullCount: StatisticCountResponse;
	    distinctCount: StatisticCountResponse;
	    duplicateCount: StatisticCountResponse;
	    min: StatisticValueResponse;
	    max: StatisticValueResponse;

	    static createFrom(source: any = {}) {
	        return new ColumnStatisticsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.nullCount = this.convertValues(source["nullCount"], StatisticCountResponse);
	        this.distinctCount = this.convertValues(source["distinctCount"], StatisticCountResponse);
	        this.duplicateCount = this.convertValues(source["duplicateCount"], StatisticCountResponse);
	        this.min = this.convertValues(source["min"], StatisticValueResponse);
	        this.max = this.convertValues(source["max"], StatisticValueResponse);
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
	export class ColumnValueInputRequest {
	    column: string;
	    kind: string;
	    value?: string;

	    static createFrom(source: any = {}) {
	        return new ColumnValueInputRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.column = source["column"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	    }
	}
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
	export class ConnectionProfilesResponse {
	    profiles: ProfileResponse[];
	    activeConnectionProfileId?: string;

	    static createFrom(source: any = {}) {
	        return new ConnectionProfilesResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profiles = this.convertValues(source["profiles"], ProfileResponse);
	        this.activeConnectionProfileId = source["activeConnectionProfileId"];
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
	export class DatabaseColumnResponse {
	    name: string;
	    dataType: string;
	    nullable: boolean;
	    isPrimaryKey: boolean;
	    isForeignKey: boolean;
	    isUnique: boolean;

	    static createFrom(source: any = {}) {
	        return new DatabaseColumnResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.dataType = source["dataType"];
	        this.nullable = source["nullable"];
	        this.isPrimaryKey = source["isPrimaryKey"];
	        this.isForeignKey = source["isForeignKey"];
	        this.isUnique = source["isUnique"];
	    }
	}
	export class DatabaseForeignKeyResponse {
	    name: string;
	    fromTable: string;
	    fromColumns: string[];
	    toTable: string;
	    toColumns: string[];

	    static createFrom(source: any = {}) {
	        return new DatabaseForeignKeyResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.fromTable = source["fromTable"];
	        this.fromColumns = source["fromColumns"];
	        this.toTable = source["toTable"];
	        this.toColumns = source["toColumns"];
	    }
	}
	export class DatabaseTableResponse {
	    namespace: string;
	    name: string;
	    columns: DatabaseColumnResponse[];

	    static createFrom(source: any = {}) {
	        return new DatabaseTableResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.namespace = source["namespace"];
	        this.name = source["name"];
	        this.columns = this.convertValues(source["columns"], DatabaseColumnResponse);
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
	export class DatabaseSchemaResponse {
	    activeProfile: ActiveProfileResponse;
	    tables: DatabaseTableResponse[];
	    foreignKeys: DatabaseForeignKeyResponse[];

	    static createFrom(source: any = {}) {
	        return new DatabaseSchemaResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeProfile = this.convertValues(source["activeProfile"], ActiveProfileResponse);
	        this.tables = this.convertValues(source["tables"], DatabaseTableResponse);
	        this.foreignKeys = this.convertValues(source["foreignKeys"], DatabaseForeignKeyResponse);
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

	export class DeleteTableRowRequest {
	    table: string;
	    locator: ColumnValueInputRequest[];

	    static createFrom(source: any = {}) {
	        return new DeleteTableRowRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.locator = this.convertValues(source["locator"], ColumnValueInputRequest);
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
	export class TableFilterRequest {
	    column: string;
	    operator: string;
	    values: string[];

	    static createFrom(source: any = {}) {
	        return new TableFilterRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.column = source["column"];
	        this.operator = source["operator"];
	        this.values = source["values"];
	    }
	}
	export class FilterGroupRequest {
	    operator: string;
	    filters: TableFilterRequest[];
	    groups: FilterGroupRequest[];

	    static createFrom(source: any = {}) {
	        return new FilterGroupRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operator = source["operator"];
	        this.filters = this.convertValues(source["filters"], TableFilterRequest);
	        this.groups = this.convertValues(source["groups"], FilterGroupRequest);
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
	export class TableFlowStateResponse {
	    x: number;
	    y: number;
	    expanded: boolean;

	    static createFrom(source: any = {}) {
	        return new TableFlowStateResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.expanded = source["expanded"];
	    }
	}
	export class FlowStateResponse {
	    version: number;
	    tableStates: Record<string, TableFlowStateResponse>;

	    static createFrom(source: any = {}) {
	        return new FlowStateResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.tableStates = this.convertValues(source["tableStates"], TableFlowStateResponse, true);
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
	export class ForeignKeyStatisticsResponse {
	    name: string;
	    fromColumns: string[];
	    toTable: string;
	    toColumns: string[];
	    sourceRowCount: StatisticCountResponse;
	    nullCount: StatisticCountResponse;
	    referencedRowCount: StatisticCountResponse;
	    missingReferenceCount: StatisticCountResponse;

	    static createFrom(source: any = {}) {
	        return new ForeignKeyStatisticsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.fromColumns = source["fromColumns"];
	        this.toTable = source["toTable"];
	        this.toColumns = source["toColumns"];
	        this.sourceRowCount = this.convertValues(source["sourceRowCount"], StatisticCountResponse);
	        this.nullCount = this.convertValues(source["nullCount"], StatisticCountResponse);
	        this.referencedRowCount = this.convertValues(source["referencedRowCount"], StatisticCountResponse);
	        this.missingReferenceCount = this.convertValues(source["missingReferenceCount"], StatisticCountResponse);
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
	export class InsertTableRowRequest {
	    table: string;
	    values: ColumnValueInputRequest[];

	    static createFrom(source: any = {}) {
	        return new InsertTableRowRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.values = this.convertValues(source["values"], ColumnValueInputRequest);
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
	export class TableSortRequest {
	    column: string;
	    direction: string;

	    static createFrom(source: any = {}) {
	        return new TableSortRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.column = source["column"];
	        this.direction = source["direction"];
	    }
	}
	export class ListTableRowsRequest {
	    table: string;
	    page: number;
	    sort?: TableSortRequest;
	    filter?: FilterGroupRequest;

	    static createFrom(source: any = {}) {
	        return new ListTableRowsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.page = source["page"];
	        this.sort = this.convertValues(source["sort"], TableSortRequest);
	        this.filter = this.convertValues(source["filter"], FilterGroupRequest);
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

	export class VerificationScenarioSummaryResponse {
	    id: string;
	    name: string;
	    primaryTable: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new VerificationScenarioSummaryResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.primaryTable = source["primaryTable"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Response___github_com_yukihito_jokyu_DB_checker_internal_handler_wails_VerificationScenarioSummaryResponse_ {
	    data?: VerificationScenarioSummaryResponse[];
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response___github_com_yukihito_jokyu_DB_checker_internal_handler_wails_VerificationScenarioSummaryResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], VerificationScenarioSummaryResponse);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_AffectedRowsResponse_ {
	    data?: AffectedRowsResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_AffectedRowsResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], AffectedRowsResponse);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_ConnectionProfilesResponse_ {
	    data?: ConnectionProfilesResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_ConnectionProfilesResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], ConnectionProfilesResponse);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_DatabaseSchemaResponse_ {
	    data?: DatabaseSchemaResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_DatabaseSchemaResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], DatabaseSchemaResponse);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_FlowStateResponse_ {
	    data?: FlowStateResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_FlowStateResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], FlowStateResponse);
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
	export class TableCellResponse {
	    kind: string;
	    value?: string;

	    static createFrom(source: any = {}) {
	        return new TableCellResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.value = source["value"];
	    }
	}
	export class TableRowResponse {
	    cells: TableCellResponse[];

	    static createFrom(source: any = {}) {
	        return new TableRowResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cells = this.convertValues(source["cells"], TableCellResponse);
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
	export class TableRowsResponse {
	    rows: TableRowResponse[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	    sort?: TableSortRequest;
	    filter?: FilterGroupRequest;

	    static createFrom(source: any = {}) {
	        return new TableRowsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], TableRowResponse);
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.sort = this.convertValues(source["sort"], TableSortRequest);
	        this.filter = this.convertValues(source["filter"], FilterGroupRequest);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_TableRowsResponse_ {
	    data?: TableRowsResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_TableRowsResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], TableRowsResponse);
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
	export class TableStatisticsResponse {
	    table: string;
	    rowCount: StatisticCountResponse;
	    columnCount: number;
	    collectedAt?: string;
	    status: string;
	    columns: ColumnStatisticsResponse[];
	    foreignKeys: ForeignKeyStatisticsResponse[];

	    static createFrom(source: any = {}) {
	        return new TableStatisticsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.rowCount = this.convertValues(source["rowCount"], StatisticCountResponse);
	        this.columnCount = source["columnCount"];
	        this.collectedAt = source["collectedAt"];
	        this.status = source["status"];
	        this.columns = this.convertValues(source["columns"], ColumnStatisticsResponse);
	        this.foreignKeys = this.convertValues(source["foreignKeys"], ForeignKeyStatisticsResponse);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_TableStatisticsResponse_ {
	    data?: TableStatisticsResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_TableStatisticsResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], TableStatisticsResponse);
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
	export class TableStructureIndexResponse {
	    name: string;
	    columns: string[];
	    unique: boolean;
	    kind: string;

	    static createFrom(source: any = {}) {
	        return new TableStructureIndexResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.columns = source["columns"];
	        this.unique = source["unique"];
	        this.kind = source["kind"];
	    }
	}
	export class TableStructureColumnResponse {
	    name: string;
	    dataType: string;
	    nullable: boolean;
	    defaultValue?: string;
	    isPrimaryKey: boolean;
	    isForeignKey: boolean;
	    isUnique: boolean;
	    isGenerated: boolean;

	    static createFrom(source: any = {}) {
	        return new TableStructureColumnResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.dataType = source["dataType"];
	        this.nullable = source["nullable"];
	        this.defaultValue = source["defaultValue"];
	        this.isPrimaryKey = source["isPrimaryKey"];
	        this.isForeignKey = source["isForeignKey"];
	        this.isUnique = source["isUnique"];
	        this.isGenerated = source["isGenerated"];
	    }
	}
	export class TableStructureTableResponse {
	    namespace: string;
	    name: string;
	    columns: TableStructureColumnResponse[];

	    static createFrom(source: any = {}) {
	        return new TableStructureTableResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.namespace = source["namespace"];
	        this.name = source["name"];
	        this.columns = this.convertValues(source["columns"], TableStructureColumnResponse);
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
	export class TableStructureResponse {
	    table: TableStructureTableResponse;
	    foreignKeys: DatabaseForeignKeyResponse[];
	    indexes: TableStructureIndexResponse[];

	    static createFrom(source: any = {}) {
	        return new TableStructureResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = this.convertValues(source["table"], TableStructureTableResponse);
	        this.foreignKeys = this.convertValues(source["foreignKeys"], DatabaseForeignKeyResponse);
	        this.indexes = this.convertValues(source["indexes"], TableStructureIndexResponse);
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
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_TableStructureResponse_ {
	    data?: TableStructureResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_TableStructureResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], TableStructureResponse);
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
	export class VerificationScenarioResponse {
	    id: string;
	    name: string;
	    primaryTable: string;
	    definition: Record<string, any>;
	    workspaceName?: string;
	    createdAt: string;
	    updatedAt: string;
	    latestRun: any;

	    static createFrom(source: any = {}) {
	        return new VerificationScenarioResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.primaryTable = source["primaryTable"];
	        this.definition = source["definition"];
	        this.workspaceName = source["workspaceName"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.latestRun = source["latestRun"];
	    }
	}
	export class Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_VerificationScenarioResponse_ {
	    data?: VerificationScenarioResponse;
	    error?: ErrorResponse;

	    static createFrom(source: any = {}) {
	        return new Response_github_com_yukihito_jokyu_DB_checker_internal_handler_wails_VerificationScenarioResponse_(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], VerificationScenarioResponse);
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
	export class SaveConnectionProfileRequest {
	    id: string;
	    name: string;
	    dbType: string;
	    host: string;
	    port: number;
	    database: string;
	    schema?: string;
	    user: string;
	    password: string;

	    static createFrom(source: any = {}) {
	        return new SaveConnectionProfileRequest(source);
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
	        this.password = source["password"];
	    }
	}
	export class TableFlowStateRequest {
	    x: number;
	    y: number;
	    expanded: boolean;

	    static createFrom(source: any = {}) {
	        return new TableFlowStateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.expanded = source["expanded"];
	    }
	}
	export class SaveFlowStateRequest {
	    version: number;
	    tableStates: Record<string, TableFlowStateRequest>;

	    static createFrom(source: any = {}) {
	        return new SaveFlowStateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.tableStates = this.convertValues(source["tableStates"], TableFlowStateRequest, true);
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















	export class UpdateTableCellRequest {
	    table: string;
	    locator: ColumnValueInputRequest[];
	    column: string;
	    value: TableCellResponse;

	    static createFrom(source: any = {}) {
	        return new UpdateTableCellRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.locator = this.convertValues(source["locator"], ColumnValueInputRequest);
	        this.column = source["column"];
	        this.value = this.convertValues(source["value"], TableCellResponse);
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

