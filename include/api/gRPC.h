#pragma once

#include "core/server/gen/libcore.pb.h"
#include <QString>

namespace QtGrpc {
    class Http2GrpcChannelPrivate;
}

namespace NekoGui_rpc {
    enum GeoRuleSetType {ip, site};

    class Client {
    public:
        explicit Client(std::function<void(const QString &)> onError, const QString &target);

        void Exit();

        // QString returns is error string

        QString Start(bool *rpcOK, const libcore::LoadConfigReq &request);

        QString Stop(bool *rpcOK);

        libcore::QueryStatsResp QueryStats();

        libcore::TestResp Test(bool *rpcOK, const libcore::TestReq &request);

        void StopTests(bool *rpcOK);

        QStringList GetGeoList(bool *rpcOK, GeoRuleSetType mode, const QString& basePath);

        QString CompileGeoSet(bool *rpcOK, GeoRuleSetType mode, std::string category, const QString& basePath);

        bool GetDNSDHCPStatus(bool *rpcOK) const;

        QString SetSystemDNS(bool *rpcOK, const QString& customNS, bool dhcp, bool clear) const;

        libcore::ListConnectionsResp ListConnections(bool *rpcOK) const;

        QString CheckConfig(bool *rpcOK, const QString& config) const;

        bool IsPrivileged(bool *rpcOK) const;

    private:
        std::function<std::unique_ptr<QtGrpc::Http2GrpcChannelPrivate>()> make_grpc_channel;
        std::unique_ptr<QtGrpc::Http2GrpcChannelPrivate> default_grpc_channel;
        std::function<void(const QString &)> onError;
    };

    inline Client *defaultClient;
} // namespace NekoGui_rpc
