import json
from requests import get
from socket import gethostbyname 


def getip(endpoint):
    #endpoint = 'https://ipinfo.io/json'
    response = get(endpoint, verify = True)
    if response.status_code != 200:
        return 'Status:', response.status_code
        exit()    
    data = response.json()
    
    ip = data['ip']
    if 'hostname' in data:
        hostname = data['hostname']
    else:
        hostname = 'hostname'
    if 'org' in data:
        org = data['org']
    else:
        org = 'org'
    city = data['city']
    loc = data['loc']
    country = data['country']
    region = data['region']
    
    print(f'Current IP is {ip} \nHostname is {hostname} \nOrg is {org} \nCity is {city} \nLocation is {loc} \nCountry is {country} \nRegion is {region}')

    
def get_ip_address(url):
    result = gethostbyname(url)
    return result
    

def port_scan():
    import port_scanner
    port_scanner
    
    
def trace(destination):
    import os
    import time
    from sys import platform
    print (time.strftime("\nDate: %d %B %Y"))
    if platform == 'win32':
        os.system('tracert' + ' ' + destination)
    else:
        os.system('traceroute' + ' ' + destination)
    
    
    
if __name__ == '__main__':
    print('\n\t\t--------IP_Master--------')
    choice = input('[1]Device IP  \t\t[2]IP lookup \t\t[3]Website IP \n[4]Traceroute \t\t[5]Port scan \n: ')
    
    if choice == '1':
        data = getip('https://ipinfo.io/json')
    
    elif choice == '2':
        target = input('Input IP for lookup: ')
        data = getip(f'https://ipinfo.io/{target}/json')
    
    elif choice == '3':
        url = input('Enter website name: ')
        result = get_ip_address(url)
        print(f'IP obtained {result}\n')
        data = getip(f'https://ipinfo.io/{result}/json')
    
    elif choice == '4':
        loc = input('Enter destination: ')
        trace(loc)
        
    elif choice == '5':
        port_scan()