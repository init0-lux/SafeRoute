import React from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity } from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import ReportCard from '@/components/ReportCard';
import TimelineItem from '@/components/TimelineItem';
import ScrollBrowseAnimation from '@/components/ScrollBrowseAnimation';

const MOCK_REPORTS = [
    { name: 'report name', status: 'active' as const, color: Colors.sosRed },
    { name: 'report name', status: 'closed' as const, color: Colors.orange },
    { name: 'report name', status: 'pending' as const, color: Colors.purple },
    { name: 'report name', status: 'active' as const, color: Colors.sosRed },
];

const MOCK_TIMELINE = [
    {
        title: 'INCIDENT RECEIVED',
        date: 'Mar 30, 5:10 PM',
        description: 'Incident was received by your nearest police station',
        icon: 'time-outline' as const,
        titleColor: Colors.red,
    },
    {
        title: 'PROGRESS UPDATE',
        date: 'Mar 30, 5:10 PM',
        description: 'Incident was received by your nearest police station',
        icon: 'eye-outline' as const,
        titleColor: Colors.red,
    },
    {
        title: 'PROGRESS UPDATE',
        date: 'Mar 30, 5:10 PM',
        description: 'Incident was received by your nearest police station',
        icon: 'eye-outline' as const,
        titleColor: Colors.red,
    },
];

export default function HistoryScreen() {
    return (
        <View style={styles.container}>
            {/* Header */}
            <View style={styles.header}>
                <TouchableOpacity onPress={() => router.back()} activeOpacity={0.7}>
                    <Ionicons name="arrow-back" size={24} color={Colors.white} />
                </TouchableOpacity>
                <Text style={styles.headerTitle}>reported incidents</Text>
                <View style={{ width: 24 }} />
            </View>

            <ScrollView
                contentContainerStyle={styles.scrollContent}
                showsVerticalScrollIndicator={false}
            >
                {/* Horizontally scrollable report cards with browsing animation */}
                <ScrollBrowseAnimation>
                    {MOCK_REPORTS.map((report, index) => (
                        <TouchableOpacity
                            key={index}
                            activeOpacity={0.8}
                            style={[
                                styles.reportCardWrapper,
                                { alignItems: index % 2 === 0 ? 'flex-start' : 'flex-end' },
                            ]}
                        >
                            <ReportCard
                                name={report.name}
                                status={report.status}
                                color={report.color}
                            />
                        </TouchableOpacity>
                    ))}
                </ScrollBrowseAnimation>

                {/* Incident details */}
                <View style={styles.detailsSection}>
                    <Text style={styles.detailLabel}>
                        <Text style={styles.detailBold}>Nature of event: </Text>
                    </Text>
                    <Text style={styles.detailLabel}>
                        <Text style={styles.detailBold}>Incident Details: </Text>
                    </Text>
                    <Text style={styles.detailLabel}>
                        <Text style={styles.detailBold}>Evidence submitted: </Text>
                    </Text>
                    <Text style={styles.detailLabel}>
                        <Text style={styles.detailBold}>Updates: </Text>
                    </Text>
                </View>

                {/* Timeline */}
                <View style={styles.timelineSection}>
                    {MOCK_TIMELINE.map((item, index) => (
                        <TimelineItem
                            key={index}
                            title={item.title}
                            date={item.date}
                            description={item.description}
                            icon={item.icon}
                            titleColor={item.titleColor}
                            isLast={index === MOCK_TIMELINE.length - 1}
                        />
                    ))}
                </View>
            </ScrollView>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.bgDark,
    },
    header: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: Spacing.lg,
        paddingTop: 60,
        paddingBottom: Spacing.lg,
    },
    headerTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '800',
        color: Colors.white,
    },
    scrollContent: {
        paddingBottom: 120,
    },
    reportCardWrapper: {
        paddingVertical: Spacing.sm,
        minHeight: 80,
    },
    detailsSection: {
        paddingHorizontal: Spacing.lg,
        marginTop: Spacing.xl,
        gap: Spacing.sm,
    },
    detailLabel: {
        fontSize: FontSizes.md,
        color: Colors.white,
        lineHeight: FontSizes.md * 1.6,
    },
    detailBold: {
        fontWeight: '700',
    },
    timelineSection: {
        paddingHorizontal: Spacing.lg,
        marginTop: Spacing.xl,
    },
});
