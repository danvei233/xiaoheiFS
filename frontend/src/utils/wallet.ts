export function normalizeWallet(data: any) {
  if (!data) return {};
  const wallet = data.wallet || data;
  return {
    balance: wallet.balance || 0,
    currency: wallet.currency || "CNY",
    updated_at: wallet.updated_at
  };
}
