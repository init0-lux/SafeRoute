import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';

export type ReportStatus = 'active' | 'closed' | 'pending';

interface ReportCardProps {
    name: string;
    status: ReportStatus;
    color?: string;
}

const STATUS_CONFIG: Record<ReportStatus, { label: string; dotColor: string; bgColor: string }> = {
    active: { label: 'active report', dotColor: '#4CAF50', bgColor: Colors.sosRed },
    closed: { label: 'closed report', dotColor: '#E8873A', bgColor: Colors.orange },
    pending: { label: 'pending status', dotColor: '#FFD700', bgColor: Colors.purple },
};

export default function ReportCard({ name, status, color }: ReportCardProps) {
    const config = STATUS_CONFIG[status];
    const bgColor = color || config.bgColor;

    return (
        <View style={styles.container}>
            <View style={[styles.nameTab, { backgroundColor: bgColor }]}>
                <Text style={styles.nameText}>{name}</Text>
            </View>
            <View style={styles.statusRow}>
                <View style={[styles.statusDot, { backgroundColor: config.dotColor }]} />
                <Text style={styles.statusText}>{config.label}</Text>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        marginBottom: Spacing.sm,
    },
    nameTab: {
        paddingHorizontal: Spacing.lg,
        paddingVertical: Spacing.sm,
        borderRadius: BorderRadius.sm,
        alignSelf: 'flex-start',
        marginBottom: 4,
    },
    nameText: {
        color: Colors.white,
        fontSize: FontSizes.md,
        fontWeight: '700',
    },
    statusRow: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: Colors.white,
        paddingHorizontal: Spacing.md,
        paddingVertical: Spacing.sm,
        borderRadius: BorderRadius.full,
        alignSelf: 'flex-start',
        gap: 6,
    },
    statusDot: {
        width: 8,
        height: 8,
        borderRadius: 4,
    },
    statusText: {
        fontSize: FontSizes.sm,
        color: Colors.bgDark,
        fontWeight: '500',
    },
});
