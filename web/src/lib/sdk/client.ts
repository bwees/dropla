/**
 * OpenAPI
 * 0.0.1
 * DO NOT MODIFY - This file has been generated using oazapfts.
 * See https://www.npmjs.com/package/oazapfts
 */
import * as Oazapfts from "@oazapfts/runtime";
import * as QS from "@oazapfts/runtime/query";
export const defaults: Oazapfts.Defaults<Oazapfts.CustomHeaders> = {
    headers: {},
    baseUrl: "/"
};
const oazapfts = Oazapfts.runtime(defaults);
export const servers = {};
export type LoginDto = {
    password: string;
    username: string;
};
export type UserDto = {
    id: string;
    isAdmin: boolean;
    quotaBytes?: number;
    username: string;
};
export type HttpError = {
    /** Human readable error message */
    detail?: string;
    errors?: {
        /** Additional information about the error */
        more?: {
            [key: string]: any;
        };
        /** For example, name of the parameter that caused the error */
        name?: string;
        /** Human readable error message */
        reason?: string;
    }[];
    instance?: string;
    /** HTTP status code */
    status?: number;
    /** Short title of the error */
    title?: string;
    /** URL of the error type. Can be used to lookup the error in a documentation */
    "type"?: string;
};
export type UnknownInterface = any;
export type SetPasswordDto = {
    currentPassword?: string;
    newPassword: string;
};
export type SetupStatusDto = {
    setupRequired: boolean;
};
export type CreateUserDto = {
    isAdmin?: boolean;
    password: string;
    username: string;
};
export type InstanceConfigDto = {
    maxRequestBytes: number;
};
export type PublicShareDto = {
    files: {
        downloadPath: string;
        name: string;
        size: number;
    }[];
    mode: string;
    requiresPassword: boolean;
    state: string;
    title: string;
    totalBytes: number;
    unlocked: boolean;
    zipPath: string;
};
export type UnlockDto = {
    password: string;
};
export type FinalizeUploadDto = {
    uploadSessionId: string;
};
export type FinalizeResultDto = {
    outcomes: {
        accepted: boolean;
        name: string;
        reason: string;
        size: number;
    }[];
};
export type ShareDto = {
    createdAt: string;
    downloads: number;
    expired: boolean;
    expiresAt?: string;
    files: {
        byOwner: boolean;
        contentType: string;
        downloadPath: string;
        id: string;
        name: string;
        size: number;
        uploadedAt: string;
        uploaderLabel: string;
    }[];
    hasPassword: boolean;
    id: string;
    maxBytes?: number;
    maxDownloads?: number;
    maxFiles?: number;
    mode: string;
    notifyEnabled: boolean;
    notifyLastError: string;
    notifyLastErrorAt?: string;
    notifyUrl: string;
    password: string;
    path: string;
    publicUrl: string;
    title: string;
    totalBytes: number;
};
export type CreateShareDto = {
    expiresInDays?: number;
    maxBytes?: number;
    maxDownloads?: number;
    maxFiles?: number;
    mode: string;
    notifyUrl?: string;
    password?: string;
    title?: string;
};
export type UpdateShareDto = {
    expiresInDays?: number;
    maxBytes?: number;
    maxDownloads?: number;
    maxFiles?: number;
    notifyEnabled?: boolean;
    notifyUrl?: string;
    password?: string;
    title?: string;
};
export type StorageOverviewDto = {
    freeBytes: number;
    users: {
        quotaBytes: number;
        shareCount: number;
        usedBytes: number;
        userId: string;
        username: string;
    }[];
};
export type AccessTokenDto = {
    createdAt: string;
    id: string;
    lastUsedAt?: string;
    name: string;
};
export type CreateAccessTokenDto = {
    name: string;
};
export type AccessTokenCreatedDto = {
    command: string;
    id: string;
    name: string;
    token: string;
};
export type UsageDto = {
    quotaBytes: number;
    shareCount: number;
    usedBytes: number;
};
export type SetQuotaDto = {
    quotaBytes?: number | null;
};
/**
 * func3
 */
export function login(loginDto: LoginDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: UserDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/auth/login", oazapfts.json({
        ...opts,
        method: "POST",
        body: loginDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func4
 */
export function logout({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/auth/logout", {
        ...opts,
        method: "POST",
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func5
 */
export function getCurrentUser({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: UserDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/auth/me", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func6
 */
export function changePassword(setPasswordDto: SetPasswordDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/auth/password", oazapfts.json({
        ...opts,
        method: "POST",
        body: setPasswordDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func1
 */
export function getSetupStatus({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: SetupStatusDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/auth/setup", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func2
 */
export function setup(createUserDto: CreateUserDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: UserDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/auth/setup", oazapfts.json({
        ...opts,
        method: "POST",
        body: createUserDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func1
 */
export function getInstanceConfig({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: InstanceConfigDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/config", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func1
 */
export function getPublicShare(token: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: PublicShareDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/s/${encodeURIComponent(token)}`, {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func2
 */
export function unlockShare(token: string, unlockDto: UnlockDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: PublicShareDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/s/${encodeURIComponent(token)}/unlock`, oazapfts.json({
        ...opts,
        method: "POST",
        body: unlockDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func3
 */
export function finaliseUpload(token: string, finalizeUploadDto: FinalizeUploadDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: FinalizeResultDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/s/${encodeURIComponent(token)}/uploads/finalise`, oazapfts.json({
        ...opts,
        method: "POST",
        body: finalizeUploadDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func1
 */
export function getShares({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: ShareDto[];
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/shares/", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func2
 */
export function createShare(createShareDto: CreateShareDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 201;
        data: ShareDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/shares/", oazapfts.json({
        ...opts,
        method: "POST",
        body: createShareDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func5
 */
export function deleteShare(id: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/shares/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE",
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func3
 */
export function getShare(id: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: ShareDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/shares/${encodeURIComponent(id)}`, {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func4
 */
export function updateShare(id: string, updateShareDto: UpdateShareDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: ShareDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/shares/${encodeURIComponent(id)}`, oazapfts.json({
        ...opts,
        method: "PATCH",
        body: updateShareDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func7
 */
export function deleteShareFile(id: string, fileId: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/shares/${encodeURIComponent(id)}/files/${encodeURIComponent(fileId)}`, {
        ...opts,
        method: "DELETE",
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func6
 */
export function rotateShareLink(id: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: ShareDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/shares/${encodeURIComponent(id)}/rotate-link`, {
        ...opts,
        method: "POST",
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func10
 */
export function getStorageOverview({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: StorageOverviewDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/storage/", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func1
 */
export function getAccessTokens({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: AccessTokenDto[];
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/tokens/", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func2
 */
export function createAccessToken(createAccessTokenDto: CreateAccessTokenDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 201;
        data: AccessTokenCreatedDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/tokens/", oazapfts.json({
        ...opts,
        method: "POST",
        body: createAccessTokenDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func3
 */
export function deleteAccessToken(id: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/tokens/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE",
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func8
 */
export function getUploadSessionShare(sessionId: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: ShareDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/upload-sessions/${encodeURIComponent(sessionId)}/share`, {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func9
 */
export function getUsage({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: UsageDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/usage/", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func1
 */
export function getUsers({ accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 200;
        data: UserDto[];
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/users/", {
        ...opts,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func2
 */
export function createUser(createUserDto: CreateUserDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 201;
        data: UserDto;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>("/api/users/", oazapfts.json({
        ...opts,
        method: "POST",
        body: createUserDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func3
 */
export function deleteUser(id: string, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/users/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE",
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    }));
}
/**
 * func4
 */
export function resetUserPassword(id: string, setPasswordDto: SetPasswordDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/users/${encodeURIComponent(id)}/password`, oazapfts.json({
        ...opts,
        method: "POST",
        body: setPasswordDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
/**
 * func5
 */
export function setUserQuota(id: string, setQuotaDto: SetQuotaDto, { accept }: {
    accept?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.ok(oazapfts.fetchJson<{
        status: 204;
        data: UnknownInterface;
    } | {
        status: 400;
        data: HttpError;
    } | {
        status: 500;
        data: HttpError;
    } | {
        status: number;
    }>(`/api/users/${encodeURIComponent(id)}/quota`, oazapfts.json({
        ...opts,
        method: "PATCH",
        body: setQuotaDto,
        headers: oazapfts.mergeHeaders(opts?.headers, {
            Accept: accept
        })
    })));
}
