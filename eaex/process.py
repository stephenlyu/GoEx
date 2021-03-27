
import json
from datetime import datetime

def format_timestamp(ts):
    return datetime.fromtimestamp(ts).strftime("%Y-%m-%d %H:%M:%S")

orders = json.loads(open('T1_USDT-orders.json').read())

prev = None
for o in orders:
    if prev is None:
        prev = o
        continue
    if o['Amount'] == prev['Amount'] and o['Side'] != prev['Side']:
        # 配对成功
        prev = None
        continue
    else:
        print('id: %s amount: %s side: %s timestamp: %s' % (prev['OrderID2'], prev['Amount'], 'buy' if prev['Side'] == 1 else 'sell', format_timestamp(prev['Timestamp']/1000)))
        prev = o