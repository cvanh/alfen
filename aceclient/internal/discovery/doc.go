// Package discovery ports the installer's LAN device discovery: the mDNS
// (DNS-SD) browser and the device registry that sits on top of it.
//
// Sources (decompiled with ilspycmd 11.1):
//
//   - ACENetwork ICUNetwork/LANConnection.cs  -> LANConnection (registry, events,
//     StartSearch/StopSearch/StartBrowsing/StopBrowsing, AddManualDevice, ...)
//   - ACENetwork ICUNetwork/BaseConnection.cs -> LANConnection.Name/Devices/NetworkInterfaces
//   - ACENetwork ICUNetwork/DeviceEventArgs.cs -> DeviceEvent
//   - ACENetwork ICUNetwork/ICULanDevice.cs   -> Device (only the members discovery
//     sets: ctor, ReInitialize TXT parsing, SetHostInfo, OnDeviceRemoved, Address,
//     Protocol, IsHTTPS, Identification, DisplayNameLine2, HasSCNNetwork)
//   - Tmds.MDns.dll (the third-party mDNS library the installer uses, decompiled
//     from firmware/msi_work/files3/Tmds.MDns.dll) -> Browser (ServiceBrowser),
//     ifaceHandler (NetworkInterfaceHandler), msgReader/buildQuery
//     (DnsMessageReader/DnsMessageWriter), dnsName (Name), QueryParameters,
//     ServiceAnnouncement.
//
// Service types browsed (LANConnection.serviceTypes): "_lolo3._http._tcp" and
// "_alfen._tcp" in the ".local." domain. A device is announced through its
// SRV target host name (ServiceAnnouncement.Hostname, e.g. "<model>-<serial>"),
// SRV port (443 => HTTPS), first A/AAAA address and TXT records
// identity=, scnnetwork=, type=<sockets>.<socketType0>.<socketType1>,
// fwversion=, euaenabled=, euaconfigured= (see Device.reInitialize).
//
// Liveness model (Tmds.MDns, not RFC 6762 cache expiry): a service is removed
// when it sends a goodbye (TTL 0 PTR/SRV/TXT, or all of its host's addresses
// with TTL 0), when its interface goes away, or when it fails to answer
// QueryParameters.Robustness consecutive queries (LANConnection uses 10, with a
// 2000 ms ResponseTime; queries go out every QueryInterval = 10 s). Positive
// record TTLs are not used for expiry, exactly as in Tmds.MDns.
//
// Replaced, not ported: the installer's ArpList (IpHlpApi GetIpNetTable P/Invoke,
// used only by DlgManualIP's "Search" button to propose a 169.254.x.x neighbour
// for manual entry). Merging mDNS results with SCN (internal/scn) results is
// left to the GUI.
//
// This package has no GUI dependencies; all events are delivered on internal
// goroutines and callers marshal them onto their UI thread themselves.
package discovery
