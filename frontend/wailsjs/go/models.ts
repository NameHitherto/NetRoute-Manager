export namespace models {
	
	export class AppSettings {
	    primaryDns: string;
	    secondaryDns: string;
	    queryInterval: number;
	    enableIpv6: boolean;
	    autoStart: boolean;
	    minToTray: boolean;
	    dnsMode: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.primaryDns = source["primaryDns"];
	        this.secondaryDns = source["secondaryDns"];
	        this.queryInterval = source["queryInterval"];
	        this.enableIpv6 = source["enableIpv6"];
	        this.autoStart = source["autoStart"];
	        this.minToTray = source["minToTray"];
	        this.dnsMode = source["dnsMode"];
	    }
	}
	export class NetworkInterface {
	    id: string;
	    name: string;
	    type: string;
	    active: boolean;
	    ipv4Gateway: string;
	    ipv6Gateway: string;
	
	    static createFrom(source: any = {}) {
	        return new NetworkInterface(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.active = source["active"];
	        this.ipv4Gateway = source["ipv4Gateway"];
	        this.ipv6Gateway = source["ipv6Gateway"];
	    }
	}
	export class RouteRule {
	    id: string;
	    domain: string;
	    port: string;
	    alias: string;
	    checked: boolean;
	    resolvedIp: string;
	    lastResolvedSec: number;
	
	    static createFrom(source: any = {}) {
	        return new RouteRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.domain = source["domain"];
	        this.port = source["port"];
	        this.alias = source["alias"];
	        this.checked = source["checked"];
	        this.resolvedIp = source["resolvedIp"];
	        this.lastResolvedSec = source["lastResolvedSec"];
	    }
	}
	export class RouteRuleInput {
	    domain: string;
	    port: string;
	    alias: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteRuleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.port = source["port"];
	        this.alias = source["alias"];
	    }
	}
	export class ServiceStartResult {
	    running: boolean;
	    nicId: string;
	    rules: RouteRule[];
	
	    static createFrom(source: any = {}) {
	        return new ServiceStartResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.nicId = source["nicId"];
	        this.rules = this.convertValues(source["rules"], RouteRule);
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

