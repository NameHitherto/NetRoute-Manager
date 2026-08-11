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

}

