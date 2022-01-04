package godevman

import (
	"fmt"
	"io"
	"math/bits"
	"math/rand"
	"net"
	"time"
)

// Returns human readable UpTime string.
// "ut" - upTime as returned by snmp agent (Time Ticks)
// "lc" - last change (Time Ticks). Submit 0 to get current upTime string
func UpTimeString(ut, lc uint64) string {
	s := (ut - lc) / 100
	d := time.Duration(s) * time.Second
	t := time.Now()
	t = t.Add(-d)
	return t.Format(time.RFC3339)
}

// Returns human readable interface status string.
func IfStatStr(n int64) string {
	switch n {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	case 4:
		return "unknown"
	case 5:
		return "dormant"
	case 6:
		return "notPresent"
	case 7:
		return "lowerLayerDown"
	default:
		return fmt.Sprintf("%d", n)
	}
}

// Returns human readable interface type string based on RFC 2863 (IANAifType-MIB).
func IfTypeStr(n int64) string {
	switch n {
	case 1:
		return "other" // none of the following
	case 2:
		return "regular1822"
	case 3:
		return "hdh1822"
	case 4:
		return "ddnX25"
	case 5:
		return "rfc877x25"
	case 6:
		return "ethernetCsmacd" // for all ethernet-like interfaces, regardless of speed, as per RFC3635
	case 7:
		return "iso88023Csmacd" // Deprecated via RFC3635, ethernetCsmacd (6) should be used instead
	case 8:
		return "iso88024TokenBus"
	case 9:
		return "iso88025TokenRing"
	case 10:
		return "iso88026Man"
	case 11:
		return "starLan" // Deprecated via RFC3635, ethernetCsmacd (6) should be used instead
	case 12:
		return "proteon10Mbit"
	case 13:
		return "proteon80Mbit"
	case 14:
		return "hyperchannel"
	case 15:
		return "fddi"
	case 16:
		return "lapb"
	case 17:
		return "sdlc"
	case 18:
		return "ds1" // DS1-MIB
	case 19:
		return "e1" // Obsolete see DS1-MIB
	case 20:
		return "basicISDN" // no longer used, see also RFC2127
	case 21:
		return "primaryISDN" // no longer used, see also RFC2127
	case 22:
		return "propPointToPointSerial" // proprietary serial
	case 23:
		return "ppp"
	case 24:
		return "softwareLoopback"
	case 25:
		return "eon" // CLNP over IP
	case 26:
		return "ethernet3Mbit"
	case 27:
		return "nsip" // XNS over IP
	case 28:
		return "slip" // generic SLIP
	case 29:
		return "ultra" // ULTRA technologies
	case 30:
		return "ds3" // DS3-MIB
	case 31:
		return "sip" // SMDS, coffee
	case 32:
		return "frameRelay" // DTE only.
	case 33:
		return "rs232"
	case 34:
		return "para" // parallel-port
	case 35:
		return "arcnet" // arcnet
	case 36:
		return "arcnetPlus" // arcnet plus
	case 37:
		return "atm" // ATM cells
	case 38:
		return "miox25"
	case 39:
		return "sonet" // SONET or SDH
	case 40:
		return "x25ple"
	case 41:
		return "iso88022llc"
	case 42:
		return "localTalk"
	case 43:
		return "smdsDxi"
	case 44:
		return "frameRelayService" // FRNETSERV-MIB
	case 45:
		return "v35"
	case 46:
		return "hssi"
	case 47:
		return "hippi"
	case 48:
		return "modem" // Generic modem
	case 49:
		return "aal5" // AAL5 over ATM
	case 50:
		return "sonetPath"
	case 51:
		return "sonetVT"
	case 52:
		return "smdsIcip" // SMDS InterCarrier Interface
	case 53:
		return "propVirtual" // proprietary virtual/internal
	case 54:
		return "propMultiplexor" // proprietary multiplexing
	case 55:
		return "ieee80212" // 100BaseVG
	case 56:
		return "fibreChannel" // Fibre Channel
	case 57:
		return "hippiInterface" // HIPPI interfaces
	case 58:
		return "frameRelayInterconnect" // Obsolete, use either frameRelay(32) or frameRelayService(44).
	case 59:
		return "aflane8023" // ATM Emulated LAN for 802.3
	case 60:
		return "aflane8025" // ATM Emulated LAN for 802.5
	case 61:
		return "cctEmul" // ATM Emulated circuit
	case 62:
		return "fastEther" // Obsoleted via RFC3635, ethernetCsmacd (6) should be used instead
	case 63:
		return "isdn" // ISDN and X.25
	case 64:
		return "v11" // CCITT V.11/X.21
	case 65:
		return "v36" // CCITT V.36
	case 66:
		return "g703at64k" // CCITT G703 at 64Kbps
	case 67:
		return "g703at2mb" // Obsolete see DS1-MIB
	case 68:
		return "qllc" // SNA QLLC
	case 69:
		return "fastEtherFX" // Obsoleted via RFC3635, ethernetCsmacd (6) should be used instead
	case 70:
		return "channel" // channel
	case 71:
		return "ieee80211" // radio spread spectrum
	case 72:
		return "ibm370parChan" // IBM System 360/370 OEMI Channel
	case 73:
		return "escon" // IBM Enterprise Systems Connection
	case 74:
		return "dlsw" // Data Link Switching
	case 75:
		return "isdns" // ISDN S/T interface
	case 76:
		return "isdnu" // ISDN U interface
	case 77:
		return "lapd" // Link Access Protocol D
	case 78:
		return "ipSwitch" // IP Switching Objects
	case 79:
		return "rsrb" // Remote Source Route Bridging
	case 80:
		return "atmLogical" // ATM Logical Port
	case 81:
		return "ds0" // Digital Signal Level 0
	case 82:
		return "ds0Bundle" // group of ds0s on the same ds1
	case 83:
		return "bsc" // Bisynchronous Protocol
	case 84:
		return "async" // Asynchronous Protocol
	case 85:
		return "cnr" // Combat Net Radio
	case 86:
		return "iso88025Dtr" // ISO 802.5r DTR
	case 87:
		return "eplrs" // Ext Pos Loc Report Sys
	case 88:
		return "arap" // Appletalk Remote Access Protocol
	case 89:
		return "propCnls" // Proprietary Connectionless Protocol
	case 90:
		return "hostPad" // CCITT-ITU X.29 PAD Protocol
	case 91:
		return "termPad" // CCITT-ITU X.3 PAD Facility
	case 92:
		return "frameRelayMPI" // Multiproto Interconnect over FR
	case 93:
		return "x213" // CCITT-ITU X213
	case 94:
		return "adsl" // Asymmetric Digital Subscriber Loop
	case 95:
		return "radsl" // Rate-Adapt. Digital Subscriber Loop
	case 96:
		return "sdsl" // Symmetric Digital Subscriber Loop
	case 97:
		return "vdsl" // Very H-Speed Digital Subscrib. Loop
	case 98:
		return "iso88025CRFPInt" // ISO 802.5 CRFP
	case 99:
		return "myrinet" // Myricom Myrinet
	case 100:
		return "voiceEM" // voice recEive and transMit
	case 101:
		return "voiceFXO" // voice Foreign Exchange Office
	case 102:
		return "voiceFXS" // voice Foreign Exchange Station
	case 103:
		return "voiceEncap" // voice encapsulation
	case 104:
		return "voiceOverIp" // voice over IP encapsulation
	case 105:
		return "atmDxi" // ATM DXI
	case 106:
		return "atmFuni" // ATM FUNI
	case 107:
		return "atmIma" // ATM IMA
	case 108:
		return "pppMultilinkBundle" // PPP Multilink Bundle
	case 109:
		return "ipOverCdlc" // IBM ipOverCdlc
	case 110:
		return "ipOverClaw" // IBM Common Link Access to Workstn
	case 111:
		return "stackToStack" // IBM stackToStack
	case 112:
		return "virtualIpAddress" // IBM VIPA
	case 113:
		return "mpc" // IBM multi-protocol channel support
	case 114:
		return "ipOverAtm" // IBM ipOverAtm
	case 115:
		return "iso88025Fiber" // ISO 802.5j Fiber Token Ring
	case 116:
		return "tdlc" // IBM twinaxial data link control
	case 117:
		return "gigabitEthernet" // Obsoleted via RFC3635, ethernetCsmacd (6) should be used instead
	case 118:
		return "hdlc" // HDLC
	case 119:
		return "lapf" // LAP F
	case 120:
		return "v37" // V.37
	case 121:
		return "x25mlp" // Multi-Link Protocol
	case 122:
		return "x25huntGroup" // X25 Hunt Group
	case 123:
		return "transpHdlc" // Transp HDLC
	case 124:
		return "interleave" // Interleave channel
	case 125:
		return "fast" // Fast channel
	case 126:
		return "ip" // IP (for APPN HPR in IP networks)
	case 127:
		return "docsCableMaclayer" // CATV Mac Layer
	case 128:
		return "docsCableDownstream" // CATV Downstream interface
	case 129:
		return "docsCableUpstream" // CATV Upstream interface
	case 130:
		return "a12MppSwitch" // Avalon Parallel Processor
	case 131:
		return "tunnel" // Encapsulation interface
	case 132:
		return "coffee" // coffee pot
	case 133:
		return "ces" // Circuit Emulation Service
	case 134:
		return "atmSubInterface" // ATM Sub Interface
	case 135:
		return "l2vlan" // Layer 2 Virtual LAN using 802.1Q
	case 136:
		return "l3ipvlan" // Layer 3 Virtual LAN using IP
	case 137:
		return "l3ipxvlan" // Layer 3 Virtual LAN using IPX
	case 138:
		return "digitalPowerline" // IP over Power Lines
	case 139:
		return "mediaMailOverIp" // Multimedia Mail over IP
	case 140:
		return "dtm" // Dynamic syncronous Transfer Mode
	case 141:
		return "dcn" // Data Communications Network
	case 142:
		return "ipForward" // IP Forwarding Interface
	case 143:
		return "msdsl" // Multi-rate Symmetric DSL
	case 144:
		return "ieee1394" // IEEE1394 High Performance Serial Bus
	case 145:
		return "if-gsn" // HIPPI-6400
	case 146:
		return "dvbRccMacLayer" // DVB-RCC MAC Layer
	case 147:
		return "dvbRccDownstream" // DVB-RCC Downstream Channel
	case 148:
		return "dvbRccUpstream" // DVB-RCC Upstream Channel
	case 149:
		return "atmVirtual" // ATM Virtual Interface
	case 150:
		return "mplsTunnel" // MPLS Tunnel Virtual Interface
	case 151:
		return "srp" // Spatial Reuse Protocol
	case 152:
		return "voiceOverAtm" // Voice Over ATM
	case 153:
		return "voiceOverFrameRelay" // Voice Over Frame Relay
	case 154:
		return "idsl" // Digital Subscriber Loop over ISDN
	case 155:
		return "compositeLink" // Avici Composite Link Interface
	case 156:
		return "ss7SigLink" // SS7 Signaling Link
	case 157:
		return "propWirelessP2P" // Prop. P2P wireless interface
	case 158:
		return "frForward" // Frame Forward Interface
	case 159:
		return "rfc1483" // Multiprotocol over ATM AAL5
	case 160:
		return "usb" // USB Interface
	case 161:
		return "ieee8023adLag" // IEEE 802.3ad Link Aggregate
	case 162:
		return "bgppolicyaccounting" // BGP Policy Accounting
	case 163:
		return "frf16MfrBundle" // FRF .16 Multilink Frame Relay
	case 164:
		return "h323Gatekeeper" // H323 Gatekeeper
	case 165:
		return "h323Proxy" // H323 Voice and Video Proxy
	case 166:
		return "mpls" // MPLS
	case 167:
		return "mfSigLink" // Multi-frequency signaling link
	case 168:
		return "hdsl2" // High Bit-Rate DSL - 2nd generation
	case 169:
		return "shdsl" // Multirate HDSL2
	case 170:
		return "ds1FDL" // Facility Data Link 4Kbps on a DS1
	case 171:
		return "pos" // Packet over SONET/SDH Interface
	case 172:
		return "dvbAsiIn" // DVB-ASI Input
	case 173:
		return "dvbAsiOut" // DVB-ASI Output
	case 174:
		return "plc" // Power Line Communtications
	case 175:
		return "nfas" // Non Facility Associated Signaling
	case 176:
		return "tr008" // TR008
	case 177:
		return "gr303RDT" // Remote Digital Terminal
	case 178:
		return "gr303IDT" // Integrated Digital Terminal
	case 179:
		return "isup" // ISUP
	case 180:
		return "propDocsWirelessMaclayer" // Cisco proprietary Maclayer
	case 181:
		return "propDocsWirelessDownstream" // Cisco proprietary Downstream
	case 182:
		return "propDocsWirelessUpstream" // Cisco proprietary Upstream
	case 183:
		return "hiperlan2" // HIPERLAN Type 2 Radio Interface
	case 184:
		return "propBWAp2Mp" // PropBroadbandWirelessAccesspt2multipt, use of this iftype for IEEE 802.16 WMAN. Interfaces as per IEEE Std 802.16f is deprecated and ifType 237 should be used instead.
	case 185:
		return "sonetOverheadChannel" // SONET Overhead Channel
	case 186:
		return "digitalWrapperOverheadChannel" // Digital Wrapper
	case 187:
		return "aal2" // ATM adaptation layer 2
	case 188:
		return "radioMAC" // MAC layer over radio links
	case 189:
		return "atmRadio" // ATM over radio links
	case 190:
		return "imt" // Inter Machine Trunks
	case 191:
		return "mvl" // Multiple Virtual Lines DSL
	case 192:
		return "reachDSL" // Long Reach DSL
	case 193:
		return "frDlciEndPt" // Frame Relay DLCI End Point
	case 194:
		return "atmVciEndPt" // ATM VCI End Point
	case 195:
		return "opticalChannel" // Optical Channel
	case 196:
		return "opticalTransport" // Optical Transport
	case 197:
		return "propAtm" // Proprietary ATM
	case 198:
		return "voiceOverCable" // Voice Over Cable Interface
	case 199:
		return "infiniband" // Infiniband
	case 200:
		return "teLink" // TE Link
	case 201:
		return "q2931" // Q.2931
	case 202:
		return "virtualTg" // Virtual Trunk Group
	case 203:
		return "sipTg" // SIP Trunk Group
	case 204:
		return "sipSig" // SIP Signaling
	case 205:
		return "docsCableUpstreamChannel" // CATV Upstream Channel
	case 206:
		return "econet" // Acorn Econet
	case 207:
		return "pon155" // FSAN 155Mb Symetrical PON interface
	case 208:
		return "pon622" // FSAN622Mb Symetrical PON interface
	case 209:
		return "bridge" // Transparent bridge interface
	case 210:
		return "linegroup" // Interface common to multiple lines
	case 211:
		return "voiceEMFGD" // voice E&M Feature Group D
	case 212:
		return "voiceFGDEANA" // voice FGD Exchange Access North American
	case 213:
		return "voiceDID" // voice Direct Inward Dialing
	case 214:
		return "mpegTransport" // MPEG transport interface
	case 215:
		return "sixToFour" // 6to4 interface (DEPRECATED)
	case 216:
		return "gtp" // GTP (GPRS Tunneling Protocol)
	case 217:
		return "pdnEtherLoop1" // Paradyne EtherLoop 1
	case 218:
		return "pdnEtherLoop2" // Paradyne EtherLoop 2
	case 219:
		return "opticalChannelGroup" // Optical Channel Group
	case 220:
		return "homepna" // HomePNA ITU-T G.989
	case 221:
		return "gfp" // Generic Framing Procedure (GFP)
	case 222:
		return "ciscoISLvlan" // Layer 2 Virtual LAN using Cisco ISL
	case 223:
		return "actelisMetaLOOP" // Acteleis proprietary MetaLOOP High Speed Link
	case 224:
		return "fcipLink" // FCIP Link
	case 225:
		return "rpr" // Resilient Packet Ring Interface Type
	case 226:
		return "qam" // RF Qam Interface
	case 227:
		return "lmp" // Link Management Protocol
	case 228:
		return "cblVectaStar" // Cambridge Broadband Networks Limited VectaStar
	case 229:
		return "docsCableMCmtsDownstream" // CATV Modular CMTS Downstream Interface
	case 230:
		return "adsl2" // Asymmetric Digital Subscriber Loop Version 2 (DEPRECATED/OBSOLETED - please use adsl2plus 238 instead)
	case 231:
		return "macSecControlledIF" // MACSecControlled
	case 232:
		return "macSecUncontrolledIF" // MACSecUncontrolled
	case 233:
		return "aviciOpticalEther" // Avici Optical Ethernet Aggregate
	case 234:
		return "atmbond" // atmbond
	case 235:
		return "voiceFGDOS" // voice FGD Operator Services
	case 236:
		return "mocaVersion1" // MultiMedia over Coax Alliance (MoCA) Interface as documented in information provided privately to IANA
	case 237:
		return "ieee80216WMAN" // IEEE 802.16 WMAN interface
	case 238:
		return "adsl2plus" // Asymmetric Digital Subscriber Loop Version 2, Version 2 Plus and all variants
	case 239:
		return "dvbRcsMacLayer" // DVB-RCS MAC Layer
	case 240:
		return "dvbTdm" // DVB Satellite TDM
	case 241:
		return "dvbRcsTdma" // DVB-RCS TDMA
	case 242:
		return "x86Laps" // LAPS based on ITU-T X.86/Y.1323
	case 243:
		return "wwanPP" // 3GPP WWAN
	case 244:
		return "wwanPP2" // 3GPP2 WWAN
	case 245:
		return "voiceEBS" // voice P-phone EBS physical interface
	case 246:
		return "ifPwType" // Pseudowire interface type
	case 247:
		return "ilan" // Internal LAN on a bridge per IEEE 802.1ap
	case 248:
		return "pip" // Provider Instance Port on a bridge per IEEE 802.1ah PBB
	case 249:
		return "aluELP" // Alcatel-Lucent Ethernet Link Protection
	case 250:
		return "gpon" // Gigabit-capable passive optical networks (G-PON) as per ITU-T G.948
	case 251:
		return "vdsl2" // Very high speed digital subscriber line Version 2 (as per ITU-T Recommendation G.993.2)
	case 252:
		return "capwapDot11Profile" // WLAN Profile Interface
	case 253:
		return "capwapDot11Bss" // WLAN BSS Interface
	case 254:
		return "capwapWtpVirtualRadio" // WTP Virtual Radio Interface
	case 255:
		return "bits" // bitsport
	case 256:
		return "docsCableUpstreamRfPort" // DOCSIS CATV Upstream RF Port
	case 257:
		return "cableDownstreamRfPort" // CATV downstream RF port
	case 258:
		return "vmwareVirtualNic" // VMware Virtual Network Interface
	case 259:
		return "ieee802154" // IEEE 802.15.4 WPAN interface
	case 260:
		return "otnOdu" // OTN Optical Data Unit
	case 261:
		return "otnOtu" // OTN Optical channel Transport Unit
	case 262:
		return "ifVfiType" // VPLS Forwarding Instance Interface Type
	case 263:
		return "g9981" // G.998.1 bonded interface
	case 264:
		return "g9982" // G.998.2 bonded interface
	case 265:
		return "g9983" // G.998.3 bonded interface
	case 266:
		return "aluEpon" // Ethernet Passive Optical Networks (E-PON)
	case 267:
		return "aluEponOnu" // EPON Optical Network Unit
	case 268:
		return "aluEponPhysicalUni" // EPON physical User to Network interface
	case 269:
		return "aluEponLogicalLink" // The emulation of a point-to-point link over the EPON layer
	case 270:
		return "aluGponOnu" // GPON Optical Network Unit
	case 271:
		return "aluGponPhysicalUni" // GPON physical User to Network interface
	case 272:
		return "vmwareNicTeam" // VMware NIC Team
	case 277:
		return "docsOfdmDownstream" // CATV Downstream OFDM interface
	case 278:
		return "docsOfdmaUpstream" // CATV Upstream OFDMA interface
	case 279:
		return "gfast" // G.fast port
	case 280:
		return "sdci" // SDCI (IO-Link)
	case 281:
		return "xboxWireless" // Xbox wireless
	case 282:
		return "fastdsl" // FastDSL
	case 283:
		return "docsCableScte55d1FwdOob" // Cable SCTE 55-1 OOB Forward Channel
	case 284:
		return "docsCableScte55d1RetOob" // Cable SCTE 55-1 OOB Return Channel
	case 285:
		return "docsCableScte55d2DsOob" // Cable SCTE 55-2 OOB Downstream Channel
	case 286:
		return "docsCableScte55d2UsOob" // Cable SCTE 55-2 OOB Upstream Channel
	case 287:
		return "docsCableNdf" // Cable Narrowband Digital Forward
	case 288:
		return "docsCableNdr" // Cable Narrowband Digital Return
	case 289:
		return "ptm" // Packet Transfer Mode
	case 290:
		return "ghn" // G.hn port
	case 291:
		return "otnOtsi" // Optical Tributary Signal
	case 292:
		return "otnOtuc" // OTN OTUCn
	case 293:
		return "otnOduc" // OTN ODUC
	case 294:
		return "otnOtsig" // OTN OTUC Signal
	case 295:
		return "microwaveCarrierTermination" // air interface of a single microwave carrier
	case 296:
		return "microwaveRadioLinkTerminal" // radio link interface for one or several aggregated microwave carriers
	case 297:
		return "ieee8021axDrni" // IEEE 802.1AX Distributed Resilient Network Interface
	case 298:
		return "ax25" // AX.25 network interfaces
	case 299:
		return "ieee19061nanocom" // Nanoscale and Molecular Communication
	case 300:
		return "cpri" // Common Public Radio Interface
	case 301:
		return "omni" // Overlay Multilink Network Interface (OMNI)
	case 302:
		return "roe" // Radio over Ethernet Interface
	case 303:
		return "p2pOverLan" // Point to Point over LAN interface
	default:
		return fmt.Sprintf("%d", n)
	}
}

// Returns bitmap of bytes
func BitMap(bytes []byte) map[int]bool {
	out := make(map[int]bool)

	for i, byte := range bytes {
		rbyte := bits.Reverse8(byte)
		for pos := 0; pos < 8; pos++ {
			if (rbyte>>pos)&1 == 1 {
				out[i*8+pos+1] = true
			}
		}
	}
	return out
}

// Returns a random string of [a-z,A-Z,0-9] chars of submitted lenght
func RandomString(l int) string {
	const charBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, l)
	for i := range b {
		b[i] = charBytes[rand.Intn(len(charBytes))]
	}

	return string(b)
}

// Make TCP request (Timeout 10s)
func TcpReq(req, host, port string) ([]byte, error) {
	// connect to this socket
	con, err := net.DialTimeout("tcp", host+":"+port, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("socket connection error: %s", err.Error())
	}

	defer con.Close()

	// set deadlines
	err = con.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("socket set read timeout: %s", err.Error())
	}

	err = con.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("socket set send timeout: %s", err.Error())
	}

	// send to socket
	_, err = con.Write([]byte(req))
	if err != nil {
		return nil, fmt.Errorf("socket send error: %s", err.Error())
	}

	// listen for reply
	res, err := io.ReadAll(con)
	if err != nil {
		return res, fmt.Errorf("socket read response error: %s", err.Error())
	}

	return res, nil
}
