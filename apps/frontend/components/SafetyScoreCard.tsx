import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';

interface SafetyScoreCardProps {
    routeName?: string;
    eta?: string;
    safetyScore?: number;
    incidentCount?: number;
}

export default function SafetyScoreCard({
    routeName = 'Safest Route',
    eta = '12mins',
    safetyScore = 2,
    incidentCount = 4,
}: SafetyScoreCardProps) {
    const stars = '⭐'.repeat(safetyScore);

    return (
        <View style={styles.container}>
            <Text style={styles.routeName}>{routeName}</Text>

            <View style={styles.infoSection}>
                <Text style={styles.infoText}>safest route evaluated</Text>
                <Text style={styles.infoText}>reach destination in {eta}</Text>
            </View>

            <View style={styles.scoreSection}>
                <Text style={styles.scoreLabel}>safety score: {stars}</Text>
                <Text style={styles.incidentText}>
                    Incidents reported on this route: {incidentCount}
                </Text>
                <Text style={styles.tipsLabel}>Safety Tips:</Text>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        backgroundColor: Colors.white,
        borderRadius: BorderRadius.lg,
        padding: Spacing.lg,
        borderWidth: 2,
        borderColor: Colors.orange,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: -4 },
        shadowOpacity: 0.1,
        shadowRadius: 8,
        elevation: 5,
    },
    routeName: {
        fontSize: FontSizes.lg,
        fontWeight: '700',
        color: Colors.bgDark,
        marginBottom: Spacing.sm,
    },
    infoSection: {
        marginBottom: Spacing.md,
    },
    infoText: {
        fontSize: FontSizes.sm,
        color: Colors.grayDark,
        lineHeight: FontSizes.sm * 1.6,
    },
    scoreSection: {
        borderTopWidth: 1,
        borderTopColor: '#E0E0E0',
        paddingTop: Spacing.md,
    },
    scoreLabel: {
        fontSize: FontSizes.sm,
        color: Colors.bgDark,
        fontWeight: '600',
        marginBottom: Spacing.xs,
    },
    incidentText: {
        fontSize: FontSizes.sm,
        color: Colors.grayDark,
        marginBottom: Spacing.xs,
    },
    tipsLabel: {
        fontSize: FontSizes.sm,
        color: Colors.bgDark,
        fontWeight: '600',
    },
});
