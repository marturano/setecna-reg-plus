#!/usr/bin/env python3
"""
Analizza un getres Setecna completo e lo confronta con la copertura
dell'add-on. Filtra il rumore (FILLER, buffer stringhe FD/XFD, matrici di
config P_*), raggruppa il mancante per famiglia e segnala i parametri con
un valore "reale" (non sentinella, non zero) come candidati prioritari.

Uso:  python3 setecna_diff.py getres.json
"""
import json, re, sys
from collections import defaultdict

# --- Copertura attuale dell'add-on -----------------------------------------
STATIC = {
 "GLOBAL_ENABLE","GLOBAL_SEASON","GLOBAL_DEICING","GLOBAL_T_EXT","GLOBAL_EXPECTED_DEWP",
 "GLOBAL_ZONE_T_HYST","GLOBAL_ZONE_RH_HYST","GLOBAL_ZONE_DEICE_TRESH","GLOBAL_ACS_ENABLE",
 "GLOBAL_T_ACS","GLOBAL_SET_ACS","ACS_MAIN_OUTPUT","ACS_SET_ECONOMY","ACS_SET_COMFORT",
 "ACS_SET_HYST","ACS_SET_DELTA","LAST_UPDATE",
}
FAMILIES = {  # prefisso -> set suffissi coperti
 "Z":{"_TEMP","_OUTPUT","_ZONE_MODE","_ZONE_SET","_FORCING","_SET_CW","_SET_EW","_SET_CS","_SET_ES","_RH","_SET_RH"},
 "C":{"_TEMP","_SET"},
 "S":{"_ENABLED","_OUTPUT","_AUXOUTPUT"},
 "D":{"_OUTPUT_DEUM","_OUTPUT_RENEW","_SPEED_LOW","_SPEED_MED","_SPEED_HIGH","_SPEED_BOOST","_SPEED_ECONOMY","_SPEED_COMFORT"},
 "EM":{"_INSTANT","_ACCLO","_ACC2LO"},
 "MT":{"_MODE","_FORCING"},
 "FAIN":{"_TEMP"},"FDIN":{"_STATUS"},"FALDIN":{"_STATUS"},
 "HP":{"_TRIT","_TEXT","_TMAND","_TACS","_STATUS","_POWER","_RQ","_OEMERROR","_OEMSTATUS"},
 "OT":set(),  # gestita a parte (OT_G<n>_*)
}
SENTINELS = {255, 32768, 32769, 65280, 65535, 65036, 65486, 65324}

# --- Rumore da ignorare completamente --------------------------------------
NOISE_RE = re.compile(r"""^(
    S?FILLER_\d+ |            # padding riservato
    X?FD\d+_[0-9A-F] |        # buffer stringhe descrizioni (char per char)
    P_OUTPUTCFG_\d+_\d+ | P_AOUTPUTCFG_\d+_\d+ |  # matrice config uscite
    P_[A-Z0-9_]+ |           # mirror parametri/config del pannello
    MT\d{2,3} |              # slot programmazione calendari (config oraria)
    NVR\d_[A-Z]+ | MATH\d_[A-Z]+ | BOOL\d_[A-Z]+ | OPTI_[A-Z0-9_]+ |
    DIALSTR_[A-Z0-9_]+ | PLD_[0-9A-F] | RING_ERROR_\d+ | BUS_ERROR(_\d+)? |
    P_SIGNATURE_\d+ | DOT_RELEASE | BOOT_PAD_\d+
)$""", re.X)

def covered(pid):
    if pid in STATIC: return True
    m = re.fullmatch(r"([A-Z]+?)(\d+)(_.+)", pid)
    if m and m.group(1) in FAMILIES and m.group(3) in FAMILIES[m.group(1)]:
        return True
    return False

def family_of(pid):
    m = re.match(r"([A-Z]+?)\d", pid)
    if m: return m.group(1)
    m = re.match(r"([A-Z]+)_", pid)
    return m.group(1) if m else pid

def main(path):
    data = json.load(open(path))["Data"]
    names = {d["Id"]: d["V"] for d in data if d["Id"].startswith(("_FREEDESC","_XFREEDESC")) and str(d["V"]).strip()}

    missing_real = defaultdict(list)
    missing_zero = defaultdict(int)
    noise = 0; cov = 0
    for d in data:
        pid, v = d["Id"], d["V"]
        if isinstance(v, str):        # descrizioni testuali gia' leggibili
            continue
        if NOISE_RE.match(pid): noise += 1; continue
        if covered(pid): cov += 1; continue
        fam = family_of(pid)
        if v in SENTINELS or v == 0:
            missing_zero[fam] += 1
        else:
            missing_real[fam].append((pid, v))

    print(f"Righe totali: {len(data)}")
    print(f"  gia' coperte:      {cov}")
    print(f"  rumore ignorato:   {noise}")
    print(f"  NON coperte:       {sum(len(v) for v in missing_real.values()) + sum(missing_zero.values())}")
    print(f"    di cui con valore reale: {sum(len(v) for v in missing_real.values())}")
    print()
    print("== NOMI PERSONALIZZATI TROVATI (utilizzabili per rinominare le entita') ==")
    for k in sorted(names, key=lambda x:(len(x),x)):
        print(f"  {k:14} = {names[k]!r}")
    print()
    print("== FAMIGLIE NON COPERTE CON ALMENO UN VALORE REALE (priorita' alta) ==")
    for fam in sorted(missing_real, key=lambda f:-len(missing_real[f])):
        ex = missing_real[fam][:6]
        extra = missing_zero.get(fam,0)
        print(f"\n  [{fam}*]  {len(missing_real[fam])} con valore reale (+{extra} a 0/sentinella)")
        for pid,v in ex:
            print(f"      {pid:28} = {v}")
    print()
    print("== FAMIGLIE NON COPERTE MA TUTTE A 0/SENTINELLA (probabilmente non installate) ==")
    only_zero = [f for f in missing_zero if f not in missing_real]
    for fam in sorted(only_zero, key=lambda f:-missing_zero[f]):
        print(f"  [{fam}*]  {missing_zero[fam]} parametri, nessun valore reale")

if __name__ == "__main__":
    main(sys.argv[1])
