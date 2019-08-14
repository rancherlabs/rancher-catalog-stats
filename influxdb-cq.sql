# byIp agg 24h
CREATE CONTINUOUS QUERY "byIp_24h" ON "catalog" RESAMPLE EVERY 6h BEGIN SELECT distinct("ip") AS ip INTO "byIp_24h" FROM "requests" WHERE "uid" != '-' GROUP BY time(24h),path,ip END
# byUid_history agg 24h
CREATE CONTINUOUS QUERY "byUid_history_24h" ON "catalog" RESAMPLE EVERY 6h BEGIN SELECT distinct("uid") AS uid INTO "byUid_history_24h" FROM "requests" GROUP BY time(24h),path,uid END
# byUid agg 24h
CREATE CONTINUOUS QUERY "byUid_1h" ON "catalog" BEGIN SELECT count(distinct("uid")) AS "unique", count(distinct("ip")) AS "unique_ip", last("uid") AS "uid" INTO "byUid_1h" FROM "requests" GROUP BY time(1h),uid,path END
CREATE CONTINUOUS QUERY "byUid_24h" ON "catalog" BEGIN SELECT last("unique") AS "unique", last("unique_ip") AS "unique_ip", last("uid") AS "uid" INTO "byUid_24h" FROM "byUid_1h" GROUP BY time(24h),uid,path END
# byCountry agg 24h
CREATE CONTINUOUS QUERY "byCountry_1h" ON "catalog" BEGIN SELECT count("ip") AS total, count(distinct("ip")) AS unique INTO "byCountry_1h" FROM "requests" GROUP BY time(1h), country, country_isocode, path END
CREATE CONTINUOUS QUERY "byCountry_24h" ON "catalog" BEGIN SELECT last("total") AS total, last("unique") AS unique INTO "byCountry_24h" FROM "byCountry_1h" GROUP BY time(1d), country, country_isocode, path END
# byIp agg 24h
CREATE CONTINUOUS QUERY "byIp_24h" ON "catalog" RESAMPLE EVERY 6h BEGIN SELECT distinct("ip") AS ip INTO "byIp_24h" FROM "requests" WHERE "uid" != '-' GROUP BY time(24h),path,ip END
# byUid_history agg 24h
CREATE CONTINUOUS QUERY "byUid_history_24h" ON "catalog" RESAMPLE EVERY 6h BEGIN SELECT distinct("uid") AS uid INTO "byUid_history_24h" FROM "requests" GROUP BY time(24h),path,uid END
# byUid agg 24h
CREATE CONTINUOUS QUERY "byUid_1h" ON "catalog" BEGIN SELECT count(distinct("uid")) AS "unique", count(distinct("ip")) AS "unique_ip", last("uid") AS "uid" INTO "byUid_1h" FROM "requests" GROUP BY time(1h),uid,path END
CREATE CONTINUOUS QUERY "byUid_24h" ON "catalog" BEGIN SELECT last("unique") AS "unique", last("unique_ip") AS "unique_ip", last("uid") AS "uid" INTO "byUid_24h" FROM "byUid_1h" GROUP BY time(24h),uid,path END
# byCountry agg 24h
CREATE CONTINUOUS QUERY "byCountry_1h" ON "catalog" BEGIN SELECT count("ip") AS total, count(distinct("ip")) AS unique INTO "byCountry_1h" FROM "requests" GROUP BY time(1h), country, country_isocode, path END
CREATE CONTINUOUS QUERY "byCountry_24h" ON "catalog" BEGIN SELECT last("total") AS total, last("unique") AS unique INTO "byCountry_24h" FROM "byCountry_1h" GROUP BY time(1d), country, country_isocode, path END
