test_list = [1, 3, 5, 6, 3, 5, 6, 1]

res = []
for x in test_list:
    if x not in res:
        res.append(x)

print(res)
